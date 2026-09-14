package database

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies all pending migrations (golang-migrate, plain SQL up/down
// pairs — docs/conventions.md). Idempotent: re-running with no pending
// migrations is a no-op (ErrNoChange is swallowed).
func Migrate(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	dbName := ""
	if u, err := url.Parse(dbURL); err == nil {
		dbName = u.Path[1:]
	}

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{DatabaseName: dbName})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}