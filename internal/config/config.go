package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all application configuration, loaded from environment variables.
type Config struct {
	// Server
	ServerPort int `env:"SERVER_PORT" envDefault:"8080"`

	// Database
	DBHost     string `env:"DB_HOST,notEmpty" envDefault:"localhost"`
	DBPort     int    `env:"DB_PORT" envDefault:"5432"`
	DBUser     string `env:"DB_USER,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBName     string `env:"DB_NAME,required"`

	// Storage
	UploadDir string `env:"UPLOAD_DIR,notEmpty" envDefault:"./uploads"`

	// Python sidecar
	PythonBin string `env:"PYTHON_BIN,notEmpty" envDefault:"./python/.venv/bin/python"`

	// AcoustID (Phase 2)
	AcoustIDAPIKey string `env:"ACOUSTID_API_KEY" envDefault:""`
}

// Load reads configuration from .env (if present) and process environment variables.
// Existing CLI/system environment variables take precedence over .env entries.
func Load() (*Config, error) {
	// Attempt loading from .env; ignore error if file is not found (e.g., in container/prod)
	_ = godotenv.Load(".env")

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config from environment: %w", err)
	}

	if cfg.ServerPort < 1 || cfg.ServerPort > 65535 {
		return nil, fmt.Errorf("SERVER_PORT must be between 1 and 65535, got %d", cfg.ServerPort)
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}
