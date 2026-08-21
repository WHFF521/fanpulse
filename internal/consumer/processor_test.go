package consumer

import (
	"context"
	"github.com/example/fanpulse/internal/messaging"
	"sync"
	"testing"
	"time"
)

type memoryDedup struct {
	mu   sync.Mutex
	seen map[string]bool
}

func (m *memoryDedup) ExecuteOnce(ctx context.Context, eventID, consumer string, fn func(context.Context) error) (bool, error) {
	m.mu.Lock()
	key := consumer + ":" + eventID
	if m.seen[key] {
		m.mu.Unlock()
		return false, nil
	}
	if err := fn(ctx); err != nil {
		m.mu.Unlock()
		return false, err
	}
	m.seen[key] = true
	m.mu.Unlock()
	return true, nil
}

func TestProcessor_HandlesDuplicateOnce(t *testing.T) {
	store := &memoryDedup{seen: map[string]bool{}}
	calls := 0
	p := NewProcessor("notification", store, func(context.Context, messaging.Event) error { calls++; return nil })
	e := messaging.Event{EventID: "msg-1", EventType: "reward.claimed.v1", SchemaVersion: 1, AggregateID: "claim-1", OccurredAt: time.Now(), Data: []byte(`{}`)}
	processed, err := p.Process(context.Background(), e)
	if err != nil || !processed {
		t.Fatalf("first processed=%v err=%v", processed, err)
	}
	processed, err = p.Process(context.Background(), e)
	if err != nil || processed || calls != 1 {
		t.Fatalf("second processed=%v calls=%d err=%v", processed, calls, err)
	}
}
