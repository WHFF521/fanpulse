package messaging

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrInvalidEvent = errors.New("invalid event")

type Event struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	SchemaVersion int             `json:"schema_version"`
	AggregateID   string          `json:"aggregate_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	TraceID       string          `json:"trace_id,omitempty"`
	Data          json.RawMessage `json:"data"`
}

func NewEvent(id, eventType, aggregateID string, data any, now time.Time) (Event, error) {
	// TODO(level-09): validate envelope fields, JSON encode data, and set version 1.
	return Event{}, ErrInvalidEvent
}

func (e Event) Validate() error {
	// TODO(level-09): validate required fields and JSON payload.
	return ErrInvalidEvent
}
