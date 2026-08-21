package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBucket_BurstAndRefill(t *testing.T) {
	now := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	b := NewBucket(2, 1, now)
	if !b.Allow(now) || !b.Allow(now) || b.Allow(now) {
		t.Fatal("burst behavior is incorrect")
	}
	if !b.Allow(now.Add(time.Second)) {
		t.Fatal("one token should refill after one second")
	}
	if b.Allow(now.Add(time.Second)) {
		t.Fatal("refilled token must be consumed")
	}
}

func TestBucket_DoesNotExceedCapacity(t *testing.T) {
	now := time.Now()
	b := NewBucket(3, 100, now)
	later := now.Add(time.Hour)
	for i := 0; i < 3; i++ {
		if !b.Allow(later) {
			t.Fatalf("token %d denied", i)
		}
	}
	if b.Allow(later) {
		t.Fatal("bucket exceeded capacity")
	}
}

func TestBucket_ConcurrentLimit(t *testing.T) {
	now := time.Now()
	b := NewBucket(10, 0, now)
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Allow(now) {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := allowed.Load(); got != 10 {
		t.Fatalf("allowed = %d, want 10", got)
	}
}
