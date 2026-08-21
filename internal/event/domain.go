package event

import (
	"errors"
	"time"
)

var (
	ErrEventNotOpen  = errors.New("event is not open")
	ErrAlreadyJoined = errors.New("event already joined")
	ErrEventNotFound = errors.New("event not found")
)

type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusPublished Status = "PUBLISHED"
	StatusActive    Status = "ACTIVE"
	StatusEnded     Status = "ENDED"
)

type Event struct {
	ID                 string
	Status             Status
	RegistrationStarts time.Time
	RegistrationEnds   time.Time
}

func (e Event) CanJoin(now time.Time) error {
	// TODO(level-03): only published/active events inside [start, end) are joinable.
	return ErrEventNotOpen
}

type Participant struct {
	ID       string
	EventID  string
	UserID   string
	JoinedAt time.Time
}
