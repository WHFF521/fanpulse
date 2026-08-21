package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_ProcessesAllJobsWithinLimit(t *testing.T) {
	jobs := make([]Job, 30)
	for i := range jobs {
		jobs[i].ID = string(rune('a' + i))
	}
	var active, maxActive, done atomic.Int64
	err := Run(context.Background(), 4, jobs, func(context.Context, Job) error {
		current := active.Add(1)
		for {
			old := maxActive.Load()
			if current <= old || maxActive.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		active.Add(-1)
		done.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if done.Load() != int64(len(jobs)) {
		t.Fatalf("done = %d", done.Load())
	}
	if maxActive.Load() > 4 {
		t.Fatalf("max concurrency = %d", maxActive.Load())
	}
}

func TestRun_CancelsAfterError(t *testing.T) {
	boom := errors.New("boom")
	jobs := []Job{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	err := Run(context.Background(), 1, jobs, func(_ context.Context, j Job) error {
		if j.ID == "2" {
			return boom
		}
		return nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v", err)
	}
}
