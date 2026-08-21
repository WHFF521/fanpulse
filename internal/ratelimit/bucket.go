package ratelimit

import (
	"sync"
	"time"
)

type Bucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	refillRate float64
	last       time.Time
}

func NewBucket(capacity int, refillPerSecond float64, now time.Time) *Bucket {
	// TODO(level-08): initialize a full bucket; handle invalid arguments.
	return &Bucket{}
}

// Allow consumes one token when available.
func (b *Bucket) Allow(now time.Time) bool {
	// TODO(level-08): refill by elapsed time, cap tokens, then consume atomically.
	return false
}
