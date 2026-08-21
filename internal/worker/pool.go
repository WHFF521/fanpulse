package worker

import (
	"context"
	"errors"
)

var ErrInvalidWorkerCount = errors.New("worker count must be positive")

type Job struct{ ID string }

// Run processes jobs with at most workerCount concurrent handlers. It returns
// the first handler error and cancels remaining work.
func Run(ctx context.Context, workerCount int, jobs []Job, handler func(context.Context, Job) error) error {
	// TODO(level-10): build a bounded worker pool with cancellation and no leaks.
	return ErrInvalidWorkerCount
}
