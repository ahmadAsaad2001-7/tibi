package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ahmadAsaad2001-7/tibi/internal/identity/login"
	"github.com/ahmadAsaad2001-7/tibi/internal/identity/logout"
	"github.com/ahmadAsaad2001-7/tibi/internal/identity/refresh"
	"github.com/ahmadAsaad2001-7/tibi/internal/identity/register"
	patientscreate "github.com/ahmadAsaad2001-7/tibi/internal/patients/createprofile"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/auth"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/config"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/database"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/httpx"
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

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Fail-closed on APP_ENV mismatch.
	if cfg.IsProduction() && cfg.JWTSecret == "replace-me-with-32-bytes-of-random" {
		return errors.New("JWT_SECRET not set in production")
	}

	logLevel := slog.LevelInfo
	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Platform
	hasher := auth.NewPasswordHasher()
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTAccessTTL)
	refreshStore := auth.NewRefreshStore(db, cfg.JWTRefreshTTL)

	// Patients module — the contract implementation.
	patientsAPI := patientscreate.NewService(db)

	// Identity module — slices.
	registerSvc := register.NewService(db, hasher, issuer, refreshStore, patientsAPI)
	loginSvc := login.NewService(db, hasher, issuer, refreshStore)
	refreshSvc := refresh.NewService(db, issuer, refreshStore)
	logoutSvc := logout.NewService(refreshStore)

	r := chi.NewRouter()
	r.Use(httpx.Trace)
	r.Use(httpx.Recover)
	r.Use(httpx.RequestLog)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", register.NewHandler(registerSvc).ServeHTTP)
			r.Post("/login", login.NewHandler(loginSvc).ServeHTTP)
			r.Post("/refresh", refresh.NewHandler(refreshSvc).ServeHTTP)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAuth(issuer))
				r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
					var cmd logout.Command
					// decode...
					if err := logoutSvc.Execute(r.Context(), cmd); err != nil {
						httpx.Error(w, r, err)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				})
			})
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http_listening", "port", cfg.HTTPPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown_initiated")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
