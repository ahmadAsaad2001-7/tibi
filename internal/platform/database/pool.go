package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

// WrapPool adapts an existing *pgxpool.Pool into a *DB. Used by the
// integration test harness (slice 18d), which starts a real Postgres
// container and needs to hand services a *DB without re-pinging it.
func WrapPool(pool *pgxpool.Pool) *DB { return &DB{Pool: pool} }

func Connect(ctx context.Context, url string, log *slog.Logger) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	// SD39: tracer always on. No opt-out.
	cfg.ConnConfig.Tracer = NewTracer(log)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() { db.Pool.Close() }

// Ping verifies the pool can reach the database. Used by the readiness probe.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
