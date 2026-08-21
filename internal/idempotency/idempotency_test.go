package idempotency

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecute_ReplaysResult(t *testing.T) {
	store := NewMemoryStore()
	calls := 0
	fn := func(context.Context) (Result, error) { calls++; return Result{201, "claim-1"}, nil }
	first, replayed, err := store.Execute(context.Background(), "key-1", "hash-a", fn)
	if err != nil || replayed {
		t.Fatalf("first = %#v replayed=%v err=%v", first, replayed, err)
	}
	second, replayed, err := store.Execute(context.Background(), "key-1", "hash-a", fn)
	if err != nil || !replayed || second != first || calls != 1 {
		t.Fatalf("second=%#v replayed=%v calls=%d err=%v", second, replayed, calls, err)
	}
}

func TestExecute_RejectsKeyReuse(t *testing.T) {
	store := NewMemoryStore()
	_, _, _ = store.Execute(context.Background(), "key-1", "hash-a", func(context.Context) (Result, error) { return Result{}, nil })
	_, _, err := store.Execute(context.Background(), "key-1", "hash-b", func(context.Context) (Result, error) { return Result{}, nil })
	if !errors.Is(err, ErrKeyReused) {
		t.Fatalf("error = %v", err)
	}
}

func TestExecute_CollapsesConcurrentCalls(t *testing.T) {
	store := NewMemoryStore()
	var calls atomic.Int64
	fn := func(context.Context) (Result, error) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		return Result{201, "claim-1"}, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, _ = store.Execute(context.Background(), "same", "hash", fn) }()
	}
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("fn calls = %d, want 1", got)
	}
}
