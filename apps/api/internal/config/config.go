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
	Environment   string
	DatabaseURL   string
	Port          string
	JWTSecret     string
	GeminiAPIKey  string
	StorageDriver string
	GCSBucket     string

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
		Environment:   envOr("ENVIRONMENT", "development"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		Port:          envOr("PORT", "8080"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		StorageDriver: envOr("STORAGE_DRIVER", "local"),
		GCSBucket:     os.Getenv("GCS_BUCKET"),
		LogSQL:        os.Getenv("LOG_SQL") == "true",
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
