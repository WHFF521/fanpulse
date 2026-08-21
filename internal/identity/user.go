package identity

import (
	"errors"
	"time"
)

var ErrInvalidUser = errors.New("invalid user")

type User struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
}

// NormalizeEmail returns the canonical email representation used by unique
// checks. FanPulse deliberately supports only basic trim/lowercase semantics.
func NormalizeEmail(email string) string {
	// TODO(level-02): trim surrounding whitespace and convert to lowercase.
	return email
}

func NewUser(id, email, displayName string, now time.Time) (User, error) {
	// TODO(level-02): validate input, normalize email, and construct the entity.
	return User{}, ErrInvalidUser
}
