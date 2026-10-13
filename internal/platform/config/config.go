package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Env           string        `env:"APP_ENV"          envDefault:"development"`
	HTTPPort      string        `env:"HTTP_PORT"        envDefault:"8080"`
	DatabaseURL   string        `env:"DATABASE_URL,required"`
	JWTSecret     string        `env:"JWT_SECRET,required"`
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_TTL"   envDefault:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_TTL"  envDefault:"720h"`

	// Kashier payment provider
	KashierAPIKey        string `env:"KASHIER_API_KEY,required"`
	KashierAPIURL        string `env:"KASHIER_API_URL"        envDefault:"https://api.kashier.io"`
	KashierWebhookSecret string `env:"KASHIER_WEBHOOK_SECRET,required"`

	// WSOriginPatterns is a comma-separated list of allowed WebSocket origins.
	// Use "*" only in development. Production should list explicit hosts.
	WSOriginPatterns []string `env:"WS_ORIGIN_PATTERNS" envSeparator:"," envDefault:"*"`

	// Storage driver: "local" (dev only) or "s3".
	StorageDriver string `env:"STORAGE_DRIVER" envDefault:"local"`

	// LocalFS driver settings.
	StorageLocalRoot    string `env:"STORAGE_LOCAL_ROOT"     envDefault:"./data/files"`
	StorageLocalBaseURL string `env:"STORAGE_LOCAL_BASE_URL" envDefault:"http://localhost:8080/files"`
	StorageLocalSignKey string `env:"STORAGE_LOCAL_SIGN_KEY" envDefault:"dev-sign-key-change-me"`

	// S3 driver settings.
	S3Endpoint  string `env:"S3_ENDPOINT"`
	S3AccessKey string `env:"S3_ACCESS_KEY"`
	S3SecretKey string `env:"S3_SECRET_KEY"`
	S3Bucket    string `env:"S3_BUCKET"    envDefault:"medical-files"`
	S3UseSSL    bool   `env:"S3_USE_SSL"   envDefault:"true"`
	S3Region    string `env:"S3_REGION"`
	S3PublicURL string `env:"S3_PUBLIC_URL"`

	// Email driver: "logger" (dev only) or "smtp".
	EmailDriver string `env:"EMAIL_DRIVER" envDefault:"logger"`

	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT"    envDefault:"587"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM"`

	// AppBaseURL is the public base URL used to build password-reset and
	// email-verification links sent out by slice 15.
	AppBaseURL string `env:"APP_BASE_URL" envDefault:"http://localhost:3000"`

	// RateLimitDriver: "memory" (in-process) or "postgres" (persistent).
	// Memory is fine for single-instance deployments; postgres survives
	// restarts and is shared across instances.
	RateLimitDriver string `env:"RATE_LIMIT_DRIVER" envDefault:"memory"`

	// Retention worker (slice 18e). Opt-in: RETENTION_ENABLED=true in
	// production, disabled everywhere else. The API binary ignores these.
	RetentionEnabled       bool          `env:"RETENTION_ENABLED"        envDefault:"false"`
	RetentionClinicalYears int           `env:"RETENTION_CLINICAL_YEARS" envDefault:"7"`
	RetentionBatchSize     int           `env:"RETENTION_BATCH_SIZE"     envDefault:"1000"`
	RetentionInterval      time.Duration `env:"RETENTION_INTERVAL"       envDefault:"24h"`
}

// Load reads config from the process environment.
//
// In non-production environments, it first loads `.env` from the working
// directory if present. Real environment variables always win — godotenv
// does not override existing values, so production deployments that set
// variables via the platform are unaffected.
func Load() (*Config, error) {
	if err := maybeLoadDotEnv(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// retentionEnv is the subset of configuration the retention worker reads.
// It deliberately excludes the API-only required fields (JWT_SECRET,
// KASHIER_API_KEY, KASHIER_WEBHOOK_SECRET) so the worker can run with just
// DATABASE_URL and retention settings — no dummy values needed.
type retentionEnv struct {
	Env                    string        `env:"APP_ENV"                  envDefault:"development"`
	DatabaseURL            string        `env:"DATABASE_URL,required"`
	RetentionEnabled       bool          `env:"RETENTION_ENABLED"        envDefault:"false"`
	RetentionClinicalYears int           `env:"RETENTION_CLINICAL_YEARS" envDefault:"7"`
	RetentionBatchSize     int           `env:"RETENTION_BATCH_SIZE"     envDefault:"1000"`
	RetentionInterval      time.Duration `env:"RETENTION_INTERVAL"       envDefault:"24h"`
}

// LoadRetention reads config for the cmd/retention binary. Solves the
// shared-vs-API config split (SD53/SD57): the worker does not need JWT or
// Kashier settings, so it must not be forced to provide them.
func LoadRetention() (*Config, error) {
	if err := maybeLoadDotEnv(); err != nil {
		return nil, err
	}

	var e retentionEnv
	if err := env.Parse(&e); err != nil {
		return nil, err
	}
	return &Config{
		Env:                    e.Env,
		DatabaseURL:            e.DatabaseURL,
		RetentionEnabled:       e.RetentionEnabled,
		RetentionClinicalYears: e.RetentionClinicalYears,
		RetentionBatchSize:     e.RetentionBatchSize,
		RetentionInterval:      e.RetentionInterval,
	}, nil
}

func maybeLoadDotEnv() error {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" || appEnv == "development" {
		if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("load .env: %w", err)
		}
	}
	return nil
}

func (c *Config) IsProduction() bool  { return c.Env == "production" }
func (c *Config) IsDevelopment() bool { return c.Env == "development" }
func (c *Config) IsStaging() bool     { return c.Env == "staging" }

// Validate runs startup assertions. Fails closed.
func (c *Config) Validate() error {
	// --- Environment-independent checks ---
	// These run in EVERY environment. A config that would crash at
	// runtime must fail at startup instead.

	switch c.StorageDriver {
	case "local", "s3":
	default:
		return configError("unknown STORAGE_DRIVER: " + c.StorageDriver)
	}

	// S3 driver requires these regardless of environment. Without this,
	// development with STORAGE_DRIVER=s3 and an empty S3_ENDPOINT
	// crashes at storage.NewS3 with a cryptic minio error.
	if c.StorageDriver == "s3" {
		if c.S3Endpoint == "" {
			return configError("S3_ENDPOINT is required when STORAGE_DRIVER=s3")
		}
		if c.S3AccessKey == "" {
			return configError("S3_ACCESS_KEY is required when STORAGE_DRIVER=s3")
		}
		if c.S3SecretKey == "" {
			return configError("S3_SECRET_KEY is required when STORAGE_DRIVER=s3")
		}
		if c.S3Bucket == "" {
			return configError("S3_BUCKET is required when STORAGE_DRIVER=s3")
		}
	}

	switch c.EmailDriver {
	case "logger", "smtp":
	default:
		return configError("unknown EMAIL_DRIVER: " + c.EmailDriver)
	}
	if c.EmailDriver == "smtp" {
		if c.SMTPHost == "" {
			return configError("SMTP_HOST is required when EMAIL_DRIVER=smtp")
		}
		if c.SMTPFrom == "" {
			return configError("SMTP_FROM is required when EMAIL_DRIVER=smtp")
		}
	}

	switch c.RateLimitDriver {
	case "memory", "postgres":
	default:
		return configError("unknown RATE_LIMIT_DRIVER: " + c.RateLimitDriver)
	}

	// Retention worker settings are validated whenever the worker is on
	// (the API binary never enables it, but fails fast if someone does).
	if err := c.validateRetention(); err != nil {
		return err
	}

	// --- Production-only checks ---
	if c.IsProduction() {
		if c.JWTSecret == "replace-me-with-32-bytes-of-random" {
			return configError("JWT_SECRET not set in production")
		}
		if strings.HasPrefix(c.KashierAPIKey, "sk_test_") {
			return configError("Kashier test key set in production")
		}
		if c.StorageDriver == "local" {
			return configError("STORAGE_DRIVER=local is not permitted in production")
		}
		if c.EmailDriver == "logger" {
			return configError("EMAIL_DRIVER=logger is not permitted in production")
		}
		if c.AppBaseURL == "http://localhost:3000" {
			return configError("APP_BASE_URL must be set in production")
		}
	}

	return nil
}

type configError string

func (e configError) Error() string { return string(e) }

// validateRetention bounds-checks the retention worker settings. Separate
// from Validate so the worker can call just this without the API checks.
func (c *Config) validateRetention() error {
	if !c.RetentionEnabled {
		return nil
	}
	if c.RetentionClinicalYears < 1 || c.RetentionClinicalYears > 50 {
		return configError("RETENTION_CLINICAL_YEARS must be 1..50")
	}
	if c.RetentionBatchSize < 100 {
		return configError("RETENTION_BATCH_SIZE must be >= 100")
	}
	if c.RetentionInterval <= 0 {
		return configError("RETENTION_INTERVAL must be > 0")
	}
	return nil
}

// ValidateRetention is what cmd/retention calls instead of the full
// Validate: it wants the retention bounds without the API-only checks
// (JWT secret, storage credentials, email driver).
func (c *Config) ValidateRetention() error { return c.validateRetention() }
