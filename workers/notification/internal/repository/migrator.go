package repository

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"borrowhub/workers/notification/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrateUp applies all up migrations.
func MigrateUp(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrating up: %w", err)
	}
	return nil
}

// MigrateDown rolls back all migrations.
func MigrateDown(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrating down: %w", err)
	}
	return nil
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("loading migration files: %w", err)
	}

	targetURL := toPgxURL(databaseURL)
	m, err := migrate.NewWithSourceInstance("iofs", d, targetURL)
	if err != nil {
		return nil, fmt.Errorf("initializing migrator: %w", err)
	}

	return m, nil
}

func toPgxURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if strings.EqualFold(u.Scheme, "postgres") || strings.EqualFold(u.Scheme, "postgresql") {
		u.Scheme = "pgx5"
	}
	return u.String()
}
