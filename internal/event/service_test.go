package event

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeEventRepo struct{ event Event }

func (f fakeEventRepo) FindByID(context.Context, string) (Event, error) { return f.event, nil }

type memoryParticipants struct{ byKey map[string]Participant }

func (m *memoryParticipants) CreateOrGet(_ context.Context, p Participant) (Participant, bool, error) {
	key := p.EventID + ":" + p.UserID
	if existing, ok := m.byKey[key]; ok {
		return existing, false, nil
	}
	m.byKey[key] = p
	return p, true, nil
}

func TestEventCanJoin_Boundaries(t *testing.T) {
	start := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	e := Event{Status: StatusPublished, RegistrationStarts: start, RegistrationEnds: start.Add(time.Hour)}
	if err := e.CanJoin(start); err != nil {
		t.Fatalf("start should be inclusive: %v", err)
	}
	if err := e.CanJoin(start.Add(time.Hour)); !errors.Is(err, ErrEventNotOpen) {
		t.Fatalf("end should be exclusive: %v", err)
	}
	e.Status = StatusDraft
	if err := e.CanJoin(start); !errors.Is(err, ErrEventNotOpen) {
		t.Fatalf("draft should not be joinable: %v", err)
	}
}

func TestServiceJoin_IsIdempotent(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 30, 0, 0, time.UTC)
	e := Event{ID: "evt-1", Status: StatusPublished, RegistrationStarts: now.Add(-time.Hour), RegistrationEnds: now.Add(time.Hour)}
	repo := &memoryParticipants{byKey: map[string]Participant{}}
	svc := NewService(fakeEventRepo{e}, repo, func() time.Time { return now }, func() string { return "part-1" })

	first, created, err := svc.Join(context.Background(), "evt-1", "usr-1")
	if err != nil || !created {
		t.Fatalf("first Join() = %#v, %v, %v", first, created, err)
	}
	second, created, err := svc.Join(context.Background(), "evt-1", "usr-1")
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("second Join() = %#v, %v, %v", second, created, err)
	}
}
