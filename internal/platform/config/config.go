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
}

// Load reads config from the process environment.
//
// In non-production environments, it first loads `.env` from the working
// directory if present. Real environment variables always win — godotenv
// does not override existing values, so production deployments that set
// variables via the platform are unaffected.
func Load() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" || appEnv == "development" {
		if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("load .env: %w", err)
		}
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool  { return c.Env == "production" }
func (c *Config) IsDevelopment() bool { return c.Env == "development" }
func (c *Config) IsStaging() bool     { return c.Env == "staging" }

// Validate runs startup assertions. Fails closed.
func (c *Config) Validate() error {
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
		if c.StorageDriver == "s3" {
			if c.S3Endpoint == "" || c.S3AccessKey == "" || c.S3SecretKey == "" {
				return configError("S3 credentials incomplete in production")
			}
		}
		// Optional: You could also add a check here to ensure WSOriginPatterns doesn't contain "*" in production
		if c.EmailDriver == "logger" {
			return configError("EMAIL_DRIVER=logger is not permitted in production")
		}
		if c.AppBaseURL == "http://localhost:3000" {
			return configError("APP_BASE_URL must be set in production")
		}
	}

	switch c.EmailDriver {
	case "logger", "smtp":
	default:
		return configError("unknown EMAIL_DRIVER: " + c.EmailDriver)
	}
	if c.EmailDriver == "smtp" {
		if c.SMTPHost == "" || c.SMTPFrom == "" {
			return configError("SMTP_HOST and SMTP_FROM are required when EMAIL_DRIVER=smtp")
		}
	}

	switch c.StorageDriver {
	case "local", "s3":
	default:
		return configError("unknown STORAGE_DRIVER: " + c.StorageDriver)
	}

	switch c.RateLimitDriver {
	case "memory", "postgres":
	default:
		return configError("unknown RATE_LIMIT_DRIVER: " + c.RateLimitDriver)
	}

	return nil
}

type configError string

func (e configError) Error() string { return string(e) }
