package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tibi/internal/platform/config"
	"tibi/internal/platform/database"
	"tibi/internal/retention"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadRetention()
	if err != nil {
		return err
	}
	if err := cfg.ValidateRetention(); err != nil {
		return err
	}
	if !cfg.RetentionEnabled {
		slog.Info("retention disabled; exiting")
		return nil
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL, slog.Default())
	if err != nil {
		return err
	}
	defer db.Close()

	worker := retention.NewWorker(db, retention.Config{
		ClinicalYears: cfg.RetentionClinicalYears,
		BatchSize:     cfg.RetentionBatchSize,
		Interval:      cfg.RetentionInterval,
	}, slog.Default())

	// Run once at startup, then on the configured interval.
	if err := worker.RunOnce(ctx); err != nil {
		slog.Error("retention run failed", "err", err)
	}

	ticker := time.NewTicker(cfg.RetentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := worker.RunOnce(ctx); err != nil {
				slog.Error("retention run failed", "err", err)
			}
		}
	}
}
