package monster

import (
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func newTestCooldownRegistry(t *testing.T) (*cooldownRegistry, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return &cooldownRegistry{client: rc}, mr
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func TestCooldown_SetAndIsOnCooldown(t *testing.T) {
	r, mr := newTestCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, byte(42), 5*time.Second)
	if !r.IsOnCooldown(ctx, tm, 100, byte(42)) {
		t.Fatalf("expected on cooldown")
	}
}

func TestCooldown_RemainingPositive(t *testing.T) {
	r, mr := newTestCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, byte(42), 5*time.Second)

	rem := r.Remaining(ctx, tm, 100, byte(42))
	if rem <= 0 || rem > 5*time.Second {
		t.Fatalf("Remaining=%s, want (0, 5s]", rem)
	}
}

func TestCooldown_RemainingMissingKeyZero(t *testing.T) {
	r, mr := newTestCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	if rem := r.Remaining(ctx, tm, 100, byte(99)); rem != 0 {
		t.Fatalf("Remaining=%s, want 0", rem)
	}
}

func TestCooldown_RemainingPastTimestampZero(t *testing.T) {
	r, mr := newTestCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	// Simulate a stale legacy value or past-expiry value by writing directly.
	key := cooldownKey(tm, 100, byte(42))
	if err := r.client.Set(ctx, key, "1", 5*time.Second).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}

	if rem := r.Remaining(ctx, tm, 100, byte(42)); rem != 0 {
		t.Fatalf("Remaining=%s, want 0 for past-timestamp value", rem)
	}
	// IsOnCooldown still uses EXISTS, so it should still report true.
	if !r.IsOnCooldown(ctx, tm, 100, byte(42)) {
		t.Fatalf("IsOnCooldown should still be true while key exists")
	}
}

func TestCooldown_ClearAll(t *testing.T) {
	r, mr := newTestCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, byte(1), time.Minute)
	r.SetCooldown(ctx, tm, 100, byte(2), time.Minute)
	r.ClearCooldowns(ctx, tm, 100)

	if r.IsOnCooldown(ctx, tm, 100, byte(1)) || r.IsOnCooldown(ctx, tm, 100, byte(2)) {
		t.Fatalf("expected all cleared")
	}
}
