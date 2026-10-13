//go:build integration

package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql
	"github.com/pressly/goose/v3"
)

// runMigrations applies the committed migrations to the test database
// using goose as a library — no goose binary required.
func runMigrations(connStr string) error {
	dir, err := findMigrationsDir()
	if err != nil {
		return err
	}
	return applyGoose(connStr, dir)
}

// findMigrationsDir walks up from the test's working directory looking for
// the <module>/migrations folder. Tests run from their package dir, so this
// is needed to locate the repository root.
func findMigrationsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("migrations dir not found from %s", dir)
		}
		dir = parent
	}
}

func applyGoose(connStr, dir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.UpContext(context.Background(), db, dir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
