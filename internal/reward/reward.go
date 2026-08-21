package reward

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrOutOfStock     = errors.New("reward out of stock")
	ErrAlreadyClaimed = errors.New("reward already claimed")
	ErrNotParticipant = errors.New("user is not an event participant")
)

// Inventory is the in-memory learning model. PostgreSQL later replaces this
// process-local lock with a conditional UPDATE inside a transaction.
type Inventory struct {
	mu        sync.Mutex
	remaining int64
	claimed   map[string]struct{}
}

func NewInventory(quantity int64) *Inventory {
	// TODO(level-06): initialize all fields and reject negative values sensibly.
	return &Inventory{}
}

func (i *Inventory) Claim(userID string) error {
	// TODO(level-06): make duplicate check + decrement one atomic critical section.
	return ErrOutOfStock
}

func (i *Inventory) Remaining() int64 {
	// TODO(level-06): read safely while claims may be running.
	return 0
}

type Claim struct{ ID, RewardID, UserID, Status string }

type ClaimRepository interface {
	ClaimAtomic(ctx context.Context, rewardID, userID, claimID string) (claim Claim, created bool, err error)
}
