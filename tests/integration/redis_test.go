//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisRewardLua_NoOversell(t *testing.T) {
	addr := os.Getenv("FANPULSE_TEST_REDIS_ADDR")
	if addr == "" {
		t.Fatal("FANPULSE_TEST_REDIS_ADDR is required for integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping Redis: %v", err)
	}
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	scriptBytes, err := os.ReadFile(filepath.Join(root, "internal", "storage", "redis", "claim.lua"))
	if err != nil {
		t.Fatalf("read claim.lua: %v", err)
	}
	stockKey := "fanpulse:test:reward:{r1}:stock"
	usersKey := "fanpulse:test:reward:{r1}:users"
	t.Cleanup(func() { client.Del(ctx, stockKey, usersKey) })
	if err := client.Set(ctx, stockKey, 10, 0).Err(); err != nil {
		t.Fatal(err)
	}

	var claimed atomic.Int64
	var wg sync.WaitGroup
	for n := 0; n < 50; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			result, err := client.Eval(ctx, string(scriptBytes), []string{stockKey, usersKey}, fmt.Sprintf("usr-%d", n)).Text()
			if err != nil {
				t.Errorf("Eval(%d): %v", n, err)
				return
			}
			switch result {
			case "CLAIMED":
				claimed.Add(1)
			case "OUT_OF_STOCK":
			default:
				t.Errorf("unexpected result %q", result)
			}
		}(n)
	}
	wg.Wait()
	if claimed.Load() != 10 {
		t.Fatalf("claimed = %d, want 10", claimed.Load())
	}
	remaining, err := client.Get(ctx, stockKey).Int()
	if err != nil || remaining != 0 {
		t.Fatalf("remaining=%d err=%v", remaining, err)
	}
	duplicate, err := client.Eval(ctx, string(scriptBytes), []string{stockKey, usersKey}, "usr-0").Text()
	if err != nil || duplicate != "DUPLICATE" {
		t.Fatalf("duplicate=%q err=%v", duplicate, err)
	}
}
