package identity

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeEmail(t *testing.T) {
	got := NormalizeEmail("  Demo.User@Example.COM ")
	if got != "demo.user@example.com" {
		t.Fatalf("NormalizeEmail() = %q", got)
	}
}

func TestNewUser(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	user, err := NewUser("usr-1", "Demo@Example.com", "Demo", now)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if user.Email != "demo@example.com" || user.CreatedAt != now {
		t.Fatalf("user = %#v", user)
	}
}

func TestNewUser_RejectsInvalidInput(t *testing.T) {
	tests := []struct{ id, email, name string }{
		{"", "demo@example.com", "Demo"},
		{"usr-1", "not-an-email", "Demo"},
		{"usr-1", "demo@example.com", ""},
	}
	for _, tt := range tests {
		_, err := NewUser(tt.id, tt.email, tt.name, time.Now())
		if !errors.Is(err, ErrInvalidUser) {
			t.Fatalf("NewUser(%q,%q,%q) error = %v", tt.id, tt.email, tt.name, err)
		}
	}
}
