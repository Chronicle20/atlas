package monster

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestAttackCooldownRegistry(t *testing.T) (*attackCooldownRegistry, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return &attackCooldownRegistry{client: rc}, mr
}

func TestAttackCooldown_SetAndIsOnCooldown(t *testing.T) {
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(1), 1500*time.Millisecond)
	if !r.IsOnCooldown(ctx, tm, 100, uint8(1)) {
		t.Fatalf("expected on cooldown for pos 1")
	}
	if r.IsOnCooldown(ctx, tm, 100, uint8(2)) {
		t.Fatalf("did not expect cooldown for pos 2")
	}
}

func TestAttackCooldown_DistinctFromSkillRegistry(t *testing.T) {
	// Sanity: same uniqueId, attack pos 0 must not collide with skill 0
	// in the OTHER registry (different key prefix). This is a simple
	// smoke test asserting different key namespaces.
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(0), 1*time.Second)
	keys := mr.Keys()
	for _, k := range keys {
		if k == "atlas:monster-cooldown:"+tm.Id().String()+":100:0" {
			t.Fatalf("attack-cooldown key collides with skill-cooldown key namespace: %s", k)
		}
	}
}

func TestAttackCooldown_ClearAll(t *testing.T) {
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(0), time.Minute)
	r.SetCooldown(ctx, tm, 100, uint8(1), time.Minute)
	r.SetCooldown(ctx, tm, 100, uint8(2), time.Minute)
	r.ClearCooldowns(ctx, tm, 100)

	if r.IsOnCooldown(ctx, tm, 100, uint8(0)) ||
		r.IsOnCooldown(ctx, tm, 100, uint8(1)) ||
		r.IsOnCooldown(ctx, tm, 100, uint8(2)) {
		t.Fatalf("expected all cleared")
	}
}

func TestAttackCooldown_ZeroDurationDoesNotPersist(t *testing.T) {
	r, mr := newTestAttackCooldownRegistry(t)
	defer mr.Close()
	ctx := context.Background()
	tm := newTestTenant(t)

	r.SetCooldown(ctx, tm, 100, uint8(0), 0)
	if r.IsOnCooldown(ctx, tm, 100, uint8(0)) {
		t.Fatalf("zero-duration cooldown must not register")
	}
}
