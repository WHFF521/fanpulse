package reward

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestInventoryClaim_PreventsDuplicate(t *testing.T) {
	inv := NewInventory(2)
	if err := inv.Claim("usr-1"); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if err := inv.Claim("usr-1"); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("duplicate error = %v", err)
	}
	if got := inv.Remaining(); got != 1 {
		t.Fatalf("remaining = %d", got)
	}
}

func TestInventoryClaim_NoOversellUnderConcurrency(t *testing.T) {
	const stock = 100
	const users = 1000
	inv := NewInventory(stock)
	var successes atomic.Int64
	var wg sync.WaitGroup
	for n := 0; n < users; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if inv.Claim(fmt.Sprintf("usr-%d", n)) == nil {
				successes.Add(1)
			}
		}(n)
	}
	wg.Wait()
	if got := successes.Load(); got != stock {
		t.Fatalf("successes = %d, want %d", got, stock)
	}
	if got := inv.Remaining(); got != 0 {
		t.Fatalf("remaining = %d", got)
	}
}
