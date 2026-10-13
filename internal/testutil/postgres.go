//go:build integration

package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var sharedPool *pgxpool.Pool

// Postgres returns the shared pool started in TestMain (see RunMain).
// One Postgres container per test binary; migrations applied once.
func Postgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if sharedPool != nil {
		return sharedPool
	}
	t.Fatal("testutil.Postgres called before TestMain — see testutil.RunMain")
	return nil
}

// RunMain is called from a TestMain. It starts a Postgres container,
// applies migrations, sets the shared pool, and runs m.Run().
func RunMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("medical"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic(err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	if err := runMigrations(connStr); err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		panic(err)
	}
	sharedPool = pool

	code := m.Run()

	pool.Close()
	_ = pgContainer.Terminate(ctx)
	os.Exit(code)
}
