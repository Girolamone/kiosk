// Command server runs the Kiosk GraphQL API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Girolamone/kiosk/apps/api/graph"
	"github.com/Girolamone/kiosk/apps/api/internal/account"
	"github.com/Girolamone/kiosk/apps/api/internal/ai"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
	"github.com/Girolamone/kiosk/apps/api/internal/config"
	"github.com/Girolamone/kiosk/apps/api/internal/db"
	"github.com/Girolamone/kiosk/apps/api/internal/httpapi"
	"github.com/Girolamone/kiosk/apps/api/internal/loaders"
	"github.com/Girolamone/kiosk/apps/api/internal/orders"
	"github.com/Girolamone/kiosk/apps/api/internal/payments"
	"github.com/Girolamone/kiosk/apps/api/internal/storage"
)

// How long a session survives before the user has to sign in again.
const sessionTTL = 7 * 24 * time.Hour

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration", "err", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	if cfg.LogSQL {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	// Cloud Run sends SIGTERM before killing an instance. Catching it lets
	// in-flight requests finish instead of failing mid-response.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Migrations run at boot. golang-migrate holds a Postgres advisory lock
	// while it works, so several instances starting at once is safe: one
	// applies the change and the others find nothing left to do.
	if err := db.Up(cfg.DatabaseURL); err != nil {
		return err
	}
	logger.Info("migrations up to date")

	var tracer pgx.QueryTracer
	if cfg.LogSQL {
		tracer = db.NewQueryLogger(logger)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL, tracer)
	if err != nil {
		return err
	}
	defer pool.Close()

	products := catalog.NewRepository(pool)
	tokens := auth.NewTokenIssuer(cfg.JWTSecret, sessionTTL)

	files, err := newStorage(cfg)
	if err != nil {
		return err
	}

	baskets := orders.NewRepository(pool)
	gateway := newPaymentGateway(cfg, logger)

	resolver := &graph.Resolver{
		Catalog:    products,
		Accounts:   account.NewService(account.NewRepository(pool)),
		Tokens:     tokens,
		Files:      files,
		Copywriter: newCopywriter(cfg, logger),
		Orders:     baskets,
		Payments:   gateway,
		PublicURL:  cfg.PublicURL,
	}

	sessions := auth.Middleware(tokens, cfg.IsProduction())
	batching := loaders.Middleware(products)

	defer func() {
		if closer, ok := files.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()

	mux := newRouter(routes{
		cfg:      cfg,
		logger:   logger,
		resolver: resolver,
		files:    files,
		gateway:  gateway,
		baskets:  baskets,
		health:   healthHandler(pool),
		sessions: sessions,
		batching: batching,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("stopped cleanly")
	return nil
}

// newCopywriter returns the configured copy generator, or one that politely
// declines. Copy generation is an accelerator: a missing key should leave the
// rest of the application working, not stop the server from starting.
func newCopywriter(cfg config.Config, logger *slog.Logger) ai.CopyGenerator {
	if cfg.GeminiAPIKey == "" {
		logger.Warn("GEMINI_API_KEY is not set, product copy generation is disabled")
		return ai.Disabled{}
	}
	logger.Info("product copy generation enabled", "model", cfg.GeminiModel)
	return ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)
}

// newPaymentGateway returns the configured gateway, or one that declines.
// A shop with no Stripe keys still lists and manages products; it just cannot
// take money.
func newPaymentGateway(cfg config.Config, logger *slog.Logger) payments.Gateway {
	if cfg.StripeSecretKey == "" {
		logger.Warn("STRIPE_SECRET_KEY is not set, checkout is disabled")
		return payments.Disabled{}
	}
	if cfg.StripeWebhookSecret == "" {
		// Without it, ParseWebhook refuses every payload, so orders would be
		// created and never settled. Better to say so at boot than to leave
		// somebody wondering why nothing is ever marked paid.
		logger.Warn("STRIPE_WEBHOOK_SECRET is not set, payments will not be recorded as settled")
	}
	logger.Info("checkout enabled")
	return payments.NewStripe(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
}

// newStorage picks the driver from configuration. Local keeps the project
// runnable with no cloud account at all.
func newStorage(cfg config.Config) (storage.Store, error) {
	switch cfg.StorageDriver {
	case "local":
		return storage.NewLocal(cfg.LocalStorageDir, "/uploads")
	case "gcs":
		return storage.NewGCS(context.Background(), cfg.GCSBucket)
	default:
		return nil, fmt.Errorf("unknown STORAGE_DRIVER %q: want \"local\" or \"gcs\"", cfg.StorageDriver)
	}
}

type middleware = func(http.Handler) http.Handler

type routes struct {
	cfg      config.Config
	logger   *slog.Logger
	resolver *graph.Resolver
	files    storage.Store
	gateway  payments.Gateway
	baskets  httpapi.OrderSettler
	health   http.Handler
	sessions middleware
	batching middleware
}

// newRouter wires every route.
//
// It is a separate function so a test can build the router without a
// database, a bucket or a network. ServeMux panics on a conflicting pattern
// at registration time, which means a duplicate route is not a bad response -
// it is a process that refuses to start, discovered on deploy. A test that
// merely calls this catches that on the laptop instead.
func newRouter(r routes) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/graphql", r.sessions(r.batching(newGraphQLHandler(r.resolver))))
	mux.Handle("POST /api/uploads", r.sessions(httpapi.Upload(r.files, r.logger)))
	// No session middleware: Stripe is the caller, and it authenticates with
	// a signature rather than a cookie.
	mux.Handle("POST /api/stripe/webhook", httpapi.StripeWebhook(r.gateway, r.baskets, r.logger))
	// Under /api and not at /healthz: Google's frontend swallows that exact
	// path on Cloud Run and answers its own 404, so the request never
	// reaches this process. Verified - every other path, /healthz/ with a
	// trailing slash included, arrives fine.
	mux.Handle("GET /api/healthz", r.health)

	// With the local driver the API serves the files it stored. With GCS the
	// bucket serves them and this route does not exist.
	if local, ok := r.files.(*storage.Local); ok {
		mux.Handle("GET /uploads/", local.Handler())
	}

	// The playground stays reachable on purpose: this is a demo, and being
	// able to explore the schema is part of the point.
	if r.cfg.WebDir != "" {
		// In production one binary serves both the API and the app, so the
		// deployment is a single origin: no CORS, and the session cookie
		// stays first-party. The app takes "/", so the playground moves.
		mux.Handle("/", httpapi.SPA(r.cfg.WebDir))
		mux.Handle("GET /playground", playground.Handler("Kiosk API", "/graphql"))
		r.logger.Info("serving the web app", "dir", r.cfg.WebDir)
	} else {
		// API only: the Vite dev server is serving the app.
		mux.Handle("/", playground.Handler("Kiosk API", "/graphql"))
	}

	return mux
}

func newGraphQLHandler(resolver *graph.Resolver) http.Handler {
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})
	return srv
}

// healthHandler reports unhealthy when the database is unreachable, so a
// broken instance is taken out of rotation instead of serving errors.
func healthHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
