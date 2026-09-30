package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Env           string        `env:"APP_ENV" envDefault:"development"`
	HTTPPort      string        `env:"HTTP_PORT" envDefault:"8080"`
	DatabaseURL   string        `env:"DATABASE_URL,required"`
	JWTSecret     string        `env:"JWT_SECRET,required"`
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
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
