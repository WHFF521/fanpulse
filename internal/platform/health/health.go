package health

import (
	"context"
	"sync"
)

type Check func(context.Context) error

type Registry struct {
	mu       sync.RWMutex
	draining bool
	checks   map[string]Check
}

func NewRegistry(checks map[string]Check) *Registry {
	// TODO(level-12): copy the input map so callers cannot mutate registry state.
	return &Registry{}
}

func (r *Registry) SetDraining(value bool) {
	// TODO(level-12): update safely.
}

func (r *Registry) Ready(ctx context.Context) (bool, map[string]string) {
	// TODO(level-12): fail while draining; run checks and return safe statuses.
	return false, nil
}
