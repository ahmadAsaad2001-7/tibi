package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost/tibi")
	t.Setenv("JWT_SECRET", "unit-test-secret")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("JWT_ACCESS_TTL", "5m")
	t.Setenv("JWT_REFRESH_TTL", "24h")
	t.Setenv("KASHIER_API_KEY", "unit-test-kashier")
	t.Setenv("KASHIER_WEBHOOK_SECRET", "unit-test-webhook")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsProduction() || cfg.IsDevelopment() || cfg.IsStaging() {
		t.Fatalf("env flags for %s", cfg.Env)
	}
	if cfg.HTTPPort != "9090" || cfg.JWTSecret != "unit-test-secret" || cfg.DatabaseURL != "postgres://localhost/tibi" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.JWTAccessTTL != 5*time.Minute || cfg.JWTRefreshTTL != 24*time.Hour {
		t.Fatalf("ttls access=%s refresh=%s", cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	}
}

func TestLoadRequiresDatabaseAndSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	unsetEnv(t, "DATABASE_URL")
	unsetEnv(t, "JWT_SECRET")
	if _, err := Load(); err == nil {
		t.Fatal("Load succeeded without required settings")
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	prev, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !ok {
			return
		}
		if err := os.Setenv(key, prev); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEnvironmentFlags(t *testing.T) {
	dev := &Config{Env: "development"}
	stage := &Config{Env: "staging"}
	if !dev.IsDevelopment() || dev.IsProduction() || !stage.IsStaging() {
		t.Fatal("environment helpers mismatch")
	}
}
