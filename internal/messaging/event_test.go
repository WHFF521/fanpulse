package messaging

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	now := time.Now().UTC()
	e, err := NewEvent("evt-msg-1", "reward.claimed.v1", "claim-1", map[string]string{"user_id": "usr-1"}, now)
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	if e.SchemaVersion != 1 || e.OccurredAt != now || !json.Valid(e.Data) {
		t.Fatalf("event = %#v", e)
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestNewEvent_RejectsMissingFields(t *testing.T) {
	_, err := NewEvent("", "reward.claimed.v1", "claim-1", struct{}{}, time.Now())
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("error = %v", err)
	}
}
