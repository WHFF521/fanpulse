package health

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryReady(t *testing.T) {
	r := NewRegistry(map[string]Check{"postgres": func(context.Context) error { return nil }})
	ready, checks := r.Ready(context.Background())
	if !ready || checks["postgres"] != "ok" {
		t.Fatalf("ready=%v checks=%v", ready, checks)
	}
	r.SetDraining(true)
	ready, checks = r.Ready(context.Background())
	if ready || checks["draining"] != "true" {
		t.Fatalf("ready=%v checks=%v", ready, checks)
	}
}

func TestRegistryReady_DependencyFailure(t *testing.T) {
	r := NewRegistry(map[string]Check{"postgres": func(context.Context) error { return errors.New("secret DSN") }})
	ready, checks := r.Ready(context.Background())
	if ready || checks["postgres"] != "unavailable" {
		t.Fatalf("ready=%v checks=%v", ready, checks)
	}
	if checks["postgres"] == "secret DSN" {
		t.Fatal("dependency detail leaked")
	}
}
