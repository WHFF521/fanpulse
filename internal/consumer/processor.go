package consumer

import (
	"context"
	"github.com/example/fanpulse/internal/messaging"
)

type DedupStore interface {
	// ExecuteOnce atomically records eventID for consumerName and invokes fn only
	// when it has not been processed. It returns processed=false for duplicates.
	ExecuteOnce(ctx context.Context, eventID, consumerName string, fn func(context.Context) error) (processed bool, err error)
}

type Processor struct {
	name    string
	store   DedupStore
	handler func(context.Context, messaging.Event) error
}

func NewProcessor(name string, store DedupStore, handler func(context.Context, messaging.Event) error) *Processor {
	return &Processor{name: name, store: store, handler: handler}
}

func (p *Processor) Process(ctx context.Context, event messaging.Event) (bool, error) {
	// TODO(level-10): validate and delegate to ExecuteOnce without swallowing errors.
	return false, nil
}
