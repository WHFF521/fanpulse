package idempotency

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrKeyReused = errors.New("idempotency key reused with another payload")
	ErrEmptyKey  = errors.New("idempotency key is required")
)

type Result struct {
	StatusCode int
	ResourceID string
}

type entry struct {
	requestHash string
	result      Result
	err         error
	done        chan struct{}
}

// MemoryStore teaches the single-flight semantics before Redis is introduced.
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]*entry
}

func NewMemoryStore() *MemoryStore {
	// TODO(level-07): initialize the store.
	return &MemoryStore{}
}

func (s *MemoryStore) Execute(ctx context.Context, key, requestHash string, fn func(context.Context) (Result, error)) (Result, bool, error) {
	// TODO(level-07): only one caller runs fn; identical callers wait and replay.
	// A different requestHash for an existing key must return ErrKeyReused.
	return Result{}, false, ErrEmptyKey
}
