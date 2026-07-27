// Command server runs the Kiosk GraphQL API.
package main

import (
	"context"
	"errors"
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Girolamone/kiosk/apps/api/graph"
	"github.com/Girolamone/kiosk/apps/api/internal/account"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
	"github.com/Girolamone/kiosk/apps/api/internal/config"
	"github.com/Girolamone/kiosk/apps/api/internal/db"
)

// How long a session survives before the user has to sign in again.
const sessionTTL = 7 * 24 * time.Hour

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

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

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, sessionTTL)
	resolver := &graph.Resolver{
		Catalog:  catalog.NewRepository(pool),
		Accounts: account.NewService(account.NewRepository(pool)),
		Tokens:   tokens,
	}

	sessions := auth.Middleware(tokens, cfg.IsProduction())

	mux := http.NewServeMux()
	mux.Handle("/graphql", sessions(newGraphQLHandler(resolver)))
	mux.Handle("GET /healthz", healthHandler(pool))
	// The playground is intentionally public: this is a demo, and being able
	// to explore the schema is the point.
	mux.Handle("/", playground.Handler("Kiosk API", "/graphql"))

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
