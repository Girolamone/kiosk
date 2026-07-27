package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Girolamone/kiosk/apps/api/graph"
	"github.com/Girolamone/kiosk/apps/api/internal/config"
	"github.com/Girolamone/kiosk/apps/api/internal/payments"
	"github.com/Girolamone/kiosk/apps/api/internal/storage"
)

func testRoutes(t *testing.T, cfg config.Config) routes {
	t.Helper()

	local, err := storage.NewLocal(t.TempDir(), "/uploads")
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	passthrough := func(next http.Handler) http.Handler { return next }

	return routes{
		cfg:      cfg,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		resolver: &graph.Resolver{},
		files:    local,
		gateway:  payments.Disabled{},
		health:   http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }),
		sessions: passthrough,
		batching: passthrough,
	}
}

// ServeMux panics on a conflicting pattern when the route is registered, so a
// duplicate route is not a bad response: it is a process that will not start.
// Building the router is enough to catch it, and it caught a real one - "/"
// registered twice once the web app started being served - that had already
// reached a deploy.
func TestRouterBuildsInEveryMode(t *testing.T) {
	t.Run("api only", func(t *testing.T) {
		newRouter(testRoutes(t, config.Config{}))
	})

	t.Run("serving the web app", func(t *testing.T) {
		newRouter(testRoutes(t, config.Config{WebDir: t.TempDir()}))
	})
}

func TestHealthIsReachable(t *testing.T) {
	router := newRouter(testRoutes(t, config.Config{}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("GET /api/healthz = %d, want 200", recorder.Code)
	}
}

// A single-page app owns its own routes, so a deep link has to come back as
// the app rather than a 404 from the file server.
func TestUnknownPathsFallBackToTheApp(t *testing.T) {
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html>app"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	router := newRouter(testRoutes(t, config.Config{WebDir: webDir}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dashboard/some-shop", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("deep link = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "app") {
		t.Errorf("body = %q, want the app's index.html", recorder.Body.String())
	}
}
