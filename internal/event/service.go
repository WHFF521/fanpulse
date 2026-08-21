package event

import (
	"context"
	"time"
)

type EventRepository interface {
	FindByID(ctx context.Context, id string) (Event, error)
}

type ParticipantRepository interface {
	// CreateOrGet must return created=false and the existing participant when
	// the (event_id,user_id) unique key already exists.
	CreateOrGet(ctx context.Context, participant Participant) (stored Participant, created bool, err error)
}

type Service struct {
	events       EventRepository
	participants ParticipantRepository
	now          func() time.Time
	newID        func() string
}

func NewService(events EventRepository, participants ParticipantRepository, now func() time.Time, newID func() string) *Service {
	return &Service{events: events, participants: participants, now: now, newID: newID}
}

func (s *Service) Join(ctx context.Context, eventID, userID string) (Participant, bool, error) {
	// TODO(level-03): load, validate, create-or-get, and preserve context errors.
	return Participant{}, false, ErrEventNotOpen
}
