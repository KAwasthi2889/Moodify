package config

import (
	"os"
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	// Set required database env vars
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "postgres")
	t.Setenv("DB_NAME", "moodify_test")
	t.Setenv("SESSION_TTL", "12h")
	t.Setenv("CLEANUP_INTERVAL", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.SessionTTL != 12*time.Hour {
		t.Errorf("expected SessionTTL 12h, got %v", cfg.SessionTTL)
	}
	if cfg.CleanupInterval != 30*time.Minute {
		t.Errorf("expected CleanupInterval 30m, got %v", cfg.CleanupInterval)
	}
}

func TestConfig_DefaultDurations(t *testing.T) {
	// Clear any overrides to test defaults
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "postgres")
	t.Setenv("DB_NAME", "moodify_test")
	_ = os.Unsetenv("SESSION_TTL")
	_ = os.Unsetenv("CLEANUP_INTERVAL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.SessionTTL != 24*time.Hour {
		t.Errorf("expected default SessionTTL 24h, got %v", cfg.SessionTTL)
	}
	if cfg.CleanupInterval != 1*time.Hour {
		t.Errorf("expected default CleanupInterval 1h, got %v", cfg.CleanupInterval)
	}
}
