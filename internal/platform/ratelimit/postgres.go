package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres persists limit state in the platform_rate_limits table so counts
// survive restarts and are shared across instances.
type Postgres struct {
	pool   *pgxpool.Pool
	cfg    Config
	log    *slog.Logger
	stopCh chan struct{}
}

func NewPostgres(pool *pgxpool.Pool, cfg Config, log *slog.Logger) *Postgres {
	p := &Postgres{
		pool:   pool,
		cfg:    cfg,
		log:    log,
		stopCh: make(chan struct{}),
	}
	go p.cleanupLoop()
	return p
}

// Allow performs a fixed-window UPSERT. The window bucket is computed in
// Go (SD38) and passed as a timestamptz.
//
// The ON CONFLICT WHERE clause makes the UPSERT a no-op when the count
// would exceed the limit, so RETURNING yields no rows -> denied.
//
// Uses pool.Exec directly, never db.Querier(ctx), so a caller's transaction
// cannot accidentally include the limiter call (SD37).
func (p *Postgres) Allow(ctx context.Context, key string) (bool, error) {
	windowStart := time.Now().Truncate(p.cfg.Window)

	var count int
	err := p.pool.QueryRow(ctx, `
		INSERT INTO platform_rate_limits (key, window_start, count)
		VALUES ($1, $2, 1)
		ON CONFLICT (key, window_start) DO UPDATE
		SET count = platform_rate_limits.count + 1
		WHERE platform_rate_limits.count < $3
		RETURNING count
	`, key, windowStart, p.cfg.Limit).Scan(&count)

	if err == pgx.ErrNoRows {
		return false, nil // over limit
	}
	if err != nil {
		return false, fmt.Errorf("ratelimit: %w", err)
	}
	return true, nil
}

// Stop terminates the cleanup goroutine.
func (p *Postgres) Stop() {
	close(p.stopCh)
}

func (p *Postgres) cleanupLoop() {
	// Clean up once per window. Rows older than 2 windows are dead.
	t := time.NewTicker(p.cfg.Window)
	defer t.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-t.C:
			cutoff := time.Now().Add(-2 * p.cfg.Window)
			_, err := p.pool.Exec(context.Background(),
				`DELETE FROM platform_rate_limits WHERE window_start < $1`,
				cutoff)
			if err != nil {
				p.log.Warn("ratelimit cleanup failed", "err", err)
			}
		}
	}
}

var _ Limiter = (*Postgres)(nil)