package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Env           string        `env:"APP_ENV" envDefault:"development"`
	HTTPPort      string        `env:"HTTP_PORT" envDefault:"8080"`
	DatabaseURL   string        `env:"DATABASE_URL,required"`
	JWTSecret     string        `env:"JWT_SECRET,required"`
	JWTAccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	JWTRefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool  { return c.Env == "production" }
func (c *Config) IsDevelopment() bool { return c.Env == "development" }
func (c *Config) IsStaging() bool     { return c.Env == "staging" }
