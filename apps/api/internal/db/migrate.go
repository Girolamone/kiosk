// Package db owns the Postgres connection and the schema migrations.
package db

import (
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the pgx5:// driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrations are compiled into the binary, so the deployed image carries its
// own schema and there is no separate migration artifact to keep in sync.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Up applies every pending migration.
func Up(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// Down rolls every migration back. Development only — it drops the schema.
func Down(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back migrations: %w", err)
	}
	return nil
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, pgxScheme(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("open migrator: %w", err)
	}
	return m, nil
}

// pgxScheme rewrites postgres:// to the scheme golang-migrate uses to pick its
// pgx v5 driver. Neon hands out postgresql:// URLs, which migrate maps to its
// older lib/pq driver.
func pgxScheme(databaseURL string) string {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return databaseURL
	}
	u.Scheme = "pgx5"
	return u.String()
}
