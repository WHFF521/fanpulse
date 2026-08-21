package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadFrom_AppliesDefaults(t *testing.T) {
	values := map[string]string{"FANPULSE_DATABASE_URL": "postgres://local/fanpulse"}
	cfg, err := LoadFrom(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Environment != "development" || cfg.HTTPAddr != ":8080" {
		t.Fatalf("defaults = env %q addr %q", cfg.Environment, cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
}

func TestLoadFrom_RejectsMissingDatabaseURL(t *testing.T) {
	_, err := LoadFrom(func(string) string { return "" })
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}

func TestLoadFrom_RejectsInvalidDuration(t *testing.T) {
	values := map[string]string{
		"FANPULSE_DATABASE_URL":     "postgres://local/fanpulse",
		"FANPULSE_SHUTDOWN_TIMEOUT": "soon",
	}
	_, err := LoadFrom(func(key string) string { return values[key] })
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}
