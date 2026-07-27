// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment     string
	DatabaseURL     string
	Port            string
	JWTSecret       string
	GeminiAPIKey    string
	StorageDriver   string
	LocalStorageDir string
	GCSBucket       string
	GeminiModel     string

	// PublicURL is where this deployment is reachable from a browser. Stripe
	// sends the shopper back here, so it has to be the real address rather
	// than anything derived from the incoming request, which a caller
	// controls.
	PublicURL           string
	StripeSecretKey     string
	StripeWebhookSecret string

	// WebDir is the built React app. Set in production, where one binary
	// serves both the app and the API from a single origin. Empty in
	// development, where the Vite dev server does it.
	WebDir string

	// LogSQL prints every statement the pool runs. Development only: it is
	// how you see an N+1 rather than guess at one.
	LogSQL bool
}

// IsProduction drives anything that has to behave differently once real
// traffic and real browsers are involved, starting with Secure cookies.
func (c Config) IsProduction() bool { return c.Environment == "production" }

// Load reads configuration and fails if anything required is missing, so a
// misconfigured deploy dies at startup instead of on the first request that
// happens to need the value.
func Load() (Config, error) {
	LoadDotEnv()

	cfg := Config{
		Environment:     envOr("ENVIRONMENT", "development"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		Port:            envOr("PORT", "8080"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		StorageDriver:   envOr("STORAGE_DRIVER", "local"),
		LocalStorageDir: envOr("LOCAL_STORAGE_DIR", "./uploads"),
		GCSBucket:       os.Getenv("GCS_BUCKET"),
		GeminiModel:     envOr("GEMINI_MODEL", "gemini-flash-latest"),
		WebDir:          os.Getenv("WEB_DIR"),
		LogSQL:          os.Getenv("LOG_SQL") == "true",

		// In development the browser talks to the Vite dev server, which
		// proxies the API, so that is the address Stripe must return to.
		PublicURL:           strings.TrimSuffix(envOr("PUBLIC_URL", "http://localhost:5173"), "/"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.StorageDriver == "gcs" && cfg.GCSBucket == "" {
		missing = append(missing, "GCS_BUCKET (required when STORAGE_DRIVER=gcs)")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing configuration: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// LoadDotEnv loads the nearest .env found walking up from the working
// directory. A missing .env is not an error: in production the values come
// from the environment directly.
func LoadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Load(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
