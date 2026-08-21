package config

import (
	"errors"
	"time"
)

var ErrInvalidConfig = errors.New("invalid configuration")

// Config contains the process configuration needed by the API and worker.
type Config struct {
	Environment     string
	HTTPAddr        string
	DatabaseURL     string
	ShutdownTimeout time.Duration
}

// LoadFrom reads configuration through getenv. Keeping getenv injectable makes
// the function deterministic in tests.
func LoadFrom(getenv func(string) string) (Config, error) {
	// TODO(level-01): parse defaults, required values, and the shutdown timeout.
	return Config{}, ErrInvalidConfig
}
