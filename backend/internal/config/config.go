package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration, sourced exclusively from environment
// variables (D-015). Env keys and defaults are documented in backend/.env.example.
type Config struct {
	DatabaseURL   string `env:"DATABASE_URL,required"`
	HTTPAddr      string `env:"HTTP_ADDR" envDefault:":8080"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`
	DefaultUserID string `env:"DEFAULT_USER_ID" envDefault:"00000000-0000-0000-0000-000000000001"`
}

// Load parses the environment. Fails on missing required vars.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	// PaaS convention (docs/conventions.md): $PORT overrides the HTTP port.
	if p := os.Getenv("PORT"); p != "" {
		cfg.HTTPAddr = ":" + p
	}
	return cfg, nil
}