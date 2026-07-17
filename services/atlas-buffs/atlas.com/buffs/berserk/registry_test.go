package berserk

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func trackedModel(characterId uint32) Model {
	return NewBuilder(world.Id(0), characterId, 10).SetChannel(channel.Id(1)).Build()
}

func TestTrackUntrackLifecycle(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))

	got, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), got.CharacterId())

	all := GetRegistry().GetAll(ctx)
	assert.Len(t, all, 1)

	tenants, err := GetRegistry().GetTenants(ctx)
	assert.NoError(t, err)
	assert.Len(t, tenants, 1, "Track must register the tenant for ticker fan-out")

	assert.NoError(t, GetRegistry().Untrack(ctx, 42))
	_, err = GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMarkDirtyAndUpdateChannelIgnoreUntracked(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	assert.NoError(t, GetRegistry().MarkDirty(ctx, 99, time.Now()))
	assert.NoError(t, GetRegistry().UpdateChannel(ctx, 99, world.Id(0), channel.Id(1)))
	assert.ErrorIs(t, GetRegistry().UpdateSkillLevel(ctx, 99, 5), ErrNotFound)
}

func TestClaimReeval(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))

	_, ok := GetRegistry().ClaimReeval(ctx, 42, now)
	assert.False(t, ok, "clean entry is not claimable")

	assert.NoError(t, GetRegistry().MarkDirty(ctx, 42, now.Add(time.Second)))
	_, ok = GetRegistry().ClaimReeval(ctx, 42, now)
	assert.False(t, ok, "grace-deferred dirty not claimable early")

	assert.NoError(t, GetRegistry().MarkDirty(ctx, 42, now))
	m, ok := GetRegistry().ClaimReeval(ctx, 42, now)
	assert.True(t, ok)
	assert.True(t, m.DirtyAt().IsZero(), "claim clears dirtyAt")

	_, ok = GetRegistry().ClaimReeval(ctx, 42, now)
	assert.False(t, ok, "second claim on same deadline loses")
}

func TestClaimReevalRequiresChannel(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	// channelKnown=false (skill-UPDATED-created entry): dirty but unroutable.
	m := NewBuilder(world.Id(0), 7, 10).SetDirtyAt(now).Build()
	assert.NoError(t, GetRegistry().Track(ctx, m))

	_, ok := GetRegistry().ClaimReeval(ctx, 7, now)
	assert.False(t, ok, "re-eval needs channelKnown for the effective-stats route")

	assert.NoError(t, GetRegistry().UpdateChannel(ctx, 7, world.Id(0), channel.Id(2)))
	_, ok = GetRegistry().ClaimReeval(ctx, 7, now)
	assert.True(t, ok, "dirtyAt survives until channel is known, then claims")
}

func TestClaimBroadcast(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))
	_, ok := GetRegistry().ClaimBroadcast(ctx, 42, now)
	assert.False(t, ok, "no broadcast before first evaluation")

	assert.NoError(t, GetRegistry().StoreEvaluation(ctx, 42, true, 120, now))
	m, ok := GetRegistry().ClaimBroadcast(ctx, 42, now)
	assert.True(t, ok)
	assert.True(t, m.Active())
	assert.Equal(t, byte(120), m.CharacterLevel())

	_, ok = GetRegistry().ClaimBroadcast(ctx, 42, now)
	assert.False(t, ok, "claim advanced the deadline by BroadcastPeriod")

	stored, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.True(t, stored.NextBroadcastAt().Equal(now.Add(BroadcastPeriod)))

	_, ok = GetRegistry().ClaimBroadcast(ctx, 42, now.Add(BroadcastPeriod))
	assert.True(t, ok, "due again one period later")
}

// TestConcurrentClaimSingleWinner is the cancel-reschedule race from the PRD's
// acceptance criteria: when two replicas scan the same due entry, exactly one
// claim wins.
func TestConcurrentClaimSingleWinner(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))
	assert.NoError(t, GetRegistry().StoreEvaluation(ctx, 42, true, 120, now))

	const attempts = 8
	wins := make(chan bool, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := GetRegistry().ClaimBroadcast(ctx, 42, now)
			wins <- ok
		}()
	}
	wg.Wait()
	close(wins)

	winners := 0
	for w := range wins {
		if w {
			winners++
		}
	}
	assert.Equal(t, 1, winners, "exactly one claimant may emit per deadline")
}

func TestTenantIsolation(t *testing.T) {
	setupTestRegistry(t)
	tenA := setupTestTenant(t)
	tenB := setupTestTenant(t)
	ctxA := setupTestContext(t, tenA)
	ctxB := setupTestContext(t, tenB)

	assert.NoError(t, GetRegistry().Track(ctxA, trackedModel(42)))

	_, err := GetRegistry().Get(ctxB, 42)
	assert.ErrorIs(t, err, ErrNotFound, "same character id in another tenant is invisible")
	assert.Empty(t, GetRegistry().GetAll(ctxB))
}
