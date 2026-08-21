//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/fanpulse/internal/event"
	"github.com/example/fanpulse/internal/reward"
	storage "github.com/example/fanpulse/internal/storage/postgres"
)

func postgresPool(t *testing.T) *storage.Store {
	t.Helper()
	url := os.Getenv("FANPULSE_TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("FANPULSE_TEST_DATABASE_URL is required for integration tests")
	}
	store, err := storage.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func migrationSQL(t *testing.T, name string) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	data, err := os.ReadFile(filepath.Join(root, "migrations", name))
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	return string(data)
}

func resetSchema(t *testing.T, store *storage.Store) {
	t.Helper()
	ctx := context.Background()
	_, _ = store.Pool().Exec(ctx, migrationSQL(t, "000002_outbox.down.sql"))
	_, _ = store.Pool().Exec(ctx, migrationSQL(t, "000001_initial_schema.down.sql"))
	if _, err := store.Pool().Exec(ctx, migrationSQL(t, "000001_initial_schema.up.sql")); err != nil {
		t.Fatalf("apply initial migration: %v", err)
	}
}

func TestPostgresMigrationAndParticipantUniqueness(t *testing.T) {
	store := postgresPool(t)
	resetSchema(t, store)
	ctx := context.Background()
	userID := "00000000-0000-0000-0000-000000000001"
	eventID := "00000000-0000-0000-0000-000000000002"
	_, err := store.Pool().Exec(ctx, `INSERT INTO users(id,email,display_name,role,created_at,updated_at) VALUES($1,'u@example.com','U','USER',now(),now())`, userID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	_, err = store.Pool().Exec(ctx, `INSERT INTO events(id,slug,title,description,status,registration_starts_at,registration_ends_at,starts_at,ends_at,created_at,updated_at) VALUES($1,'demo','Demo','','PUBLISHED',now()-interval '1 hour',now()+interval '1 hour',now()+interval '2 hours',now()+interval '3 hours',now(),now())`, eventID)
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}

	repo := storage.NewParticipantRepository(store)
	var created atomic.Int64
	var wg sync.WaitGroup
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p := event.Participant{ID: fmt.Sprintf("10000000-0000-0000-0000-%012d", n), EventID: eventID, UserID: userID, JoinedAt: time.Now().UTC()}
			_, wasCreated, err := repo.CreateOrGet(ctx, p)
			if err != nil {
				t.Errorf("CreateOrGet(%d): %v", n, err)
				return
			}
			if wasCreated {
				created.Add(1)
			}
		}(n)
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Fatalf("created = %d, want 1", created.Load())
	}
	var count int
	if err := store.Pool().QueryRow(ctx, `SELECT count(*) FROM event_participants WHERE event_id=$1 AND user_id=$2`, eventID, userID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rows = %d, want 1", count)
	}
}

func TestRewardClaimAtomic_NoOversell(t *testing.T) {
	store := postgresPool(t)
	resetSchema(t, store)
	ctx := context.Background()
	eventID := "20000000-0000-0000-0000-000000000001"
	rewardID := "20000000-0000-0000-0000-000000000002"
	_, err := store.Pool().Exec(ctx, `INSERT INTO events(id,slug,title,description,status,registration_starts_at,registration_ends_at,starts_at,ends_at,created_at,updated_at) VALUES($1,'reward-demo','Demo','','ACTIVE',now()-interval '2 hours',now()+interval '1 hour',now()-interval '1 hour',now()+interval '2 hours',now(),now())`, eventID)
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}
	_, err = store.Pool().Exec(ctx, `INSERT INTO rewards(id,event_id,name,status,total_quantity,remaining_quantity,claim_starts_at,claim_ends_at,created_at,updated_at) VALUES($1,$2,'Badge','ACTIVE',10,10,now()-interval '1 hour',now()+interval '1 hour',now(),now())`, rewardID, eventID)
	if err != nil {
		t.Fatalf("seed reward: %v", err)
	}
	for n := 0; n < 50; n++ {
		userID := fmt.Sprintf("30000000-0000-0000-0000-%012d", n)
		_, err = store.Pool().Exec(ctx, `INSERT INTO users(id,email,display_name,role,created_at,updated_at) VALUES($1,$2,$3,'USER',now(),now())`, userID, fmt.Sprintf("u%d@example.com", n), fmt.Sprintf("U%d", n))
		if err != nil {
			t.Fatalf("seed user %d: %v", n, err)
		}
	}
	repo := storage.NewClaimRepository(store)
	var successes atomic.Int64
	var wg sync.WaitGroup
	for n := 0; n < 50; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			userID := fmt.Sprintf("30000000-0000-0000-0000-%012d", n)
			claimID := fmt.Sprintf("40000000-0000-0000-0000-%012d", n)
			_, created, err := repo.ClaimAtomic(ctx, rewardID, userID, claimID)
			if err == nil && created {
				successes.Add(1)
				return
			}
			if err != nil && !errors.Is(err, reward.ErrOutOfStock) {
				t.Errorf("claim %d: %v", n, err)
			}
		}(n)
	}
	wg.Wait()
	if successes.Load() != 10 {
		t.Fatalf("successes = %d, want 10", successes.Load())
	}
	var remaining, claims int
	if err := store.Pool().QueryRow(ctx, `SELECT remaining_quantity FROM rewards WHERE id=$1`, rewardID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := store.Pool().QueryRow(ctx, `SELECT count(*) FROM reward_claims WHERE reward_id=$1 AND status='SUCCEEDED'`, rewardID).Scan(&claims); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 || claims != 10 {
		t.Fatalf("remaining=%d claims=%d", remaining, claims)
	}
}
