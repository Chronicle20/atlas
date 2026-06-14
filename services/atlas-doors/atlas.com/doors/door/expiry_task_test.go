package door

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// fakeExpiryProcessor records which owner ids were passed to RemoveByOwner.
type fakeExpiryProcessor struct {
	removed []character.Id
}

func (f *fakeExpiryProcessor) RemoveByOwner(ownerCharacterId character.Id, reason string) error {
	f.removed = append(f.removed, ownerCharacterId)
	return nil
}

// TestExpiryRemovesOnlyExpiredPastGrace seeds three doors in the miniredis-backed
// registry (reuses TestMain) and verifies only the past-grace expired door is removed.
//
//   - door1: deployTime=now-10m, expiresAt=now-1m  → removed (expired & past grace).
//   - door2: deployTime=now,     expiresAt=now-1ms  → NOT removed (within 3s grace).
//   - door3: expiresAt in future                    → NOT removed.
func TestExpiryRemovesOnlyExpiredPastGrace(t *testing.T) {
	ten, ctx := newTestTenant()
	testRegistry.Clear(ctx)

	now := time.Now()
	f := field.NewBuilder(1, 2, 100000000).Build()

	// door1: expired well past the grace window.
	door1 := NewBuilder().
		SetAreaDoorId(10_000_001).
		SetTownDoorId(10_000_002).
		SetOwnerCharacterId(1001).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(_map.Id(104000000)).
		SetSlot(0).
		SetDeployTime(now.Add(-10 * time.Minute)).
		SetExpiresAt(now.Add(-1 * time.Minute)).
		Build()

	// door2: expired (1ms ago) but deployTime is ~now → within the 3s grace.
	door2 := NewBuilder().
		SetAreaDoorId(10_000_003).
		SetTownDoorId(10_000_004).
		SetOwnerCharacterId(1002).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(_map.Id(104000000)).
		SetSlot(1).
		SetDeployTime(now).
		SetExpiresAt(now.Add(-time.Millisecond)).
		Build()

	// door3: expires in the future → not removed.
	door3 := NewBuilder().
		SetAreaDoorId(10_000_005).
		SetTownDoorId(10_000_006).
		SetOwnerCharacterId(1003).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(_map.Id(104000000)).
		SetSlot(2).
		SetDeployTime(now.Add(-5 * time.Minute)).
		SetExpiresAt(now.Add(5 * time.Minute)).
		Build()

	for _, m := range []Model{door1, door2, door3} {
		if err := testRegistry.Put(ctx, ten, m); err != nil {
			t.Fatalf("Put door %d: %v", m.AreaDoorId(), err)
		}
	}

	fake := &fakeExpiryProcessor{}
	task := &ExpiryTask{
		l:        logrus.New(),
		ctx:      ctx,
		interval: 30 * time.Second,
		newProcessor: func(l logrus.FieldLogger, tctx context.Context) expiryProcessor {
			// Verify the processor is created with a context that carries a tenant.
			if _, err := tenant.FromContext(tctx)(); err != nil {
				t.Errorf("ExpiryTask did not inject tenant into context: %v", err)
			}
			return fake
		},
	}

	task.Run()

	// Only door1's owner (1001) should have been removed.
	if len(fake.removed) != 1 {
		t.Fatalf("expected 1 removal, got %d: %v", len(fake.removed), fake.removed)
	}
	if fake.removed[0] != 1001 {
		t.Fatalf("expected owner 1001 to be removed, got %v", fake.removed)
	}
}

// TestExpiryTaskSleepTime verifies SleepTime() returns the configured interval.
func TestExpiryTaskSleepTime(t *testing.T) {
	task := &ExpiryTask{interval: 42 * time.Second}
	if task.SleepTime() != 42*time.Second {
		t.Fatalf("SleepTime: want 42s got %v", task.SleepTime())
	}
}

// TestExpirySkipsZeroExpiresAt verifies that a door with a zero ExpiresAt
// (no expiry configured) is never removed by the sweep.
func TestExpirySkipsZeroExpiresAt(t *testing.T) {
	ten, ctx := newTestTenant()
	testRegistry.Clear(ctx)

	now := time.Now()
	f := field.NewBuilder(1, 2, 100000000).Build()

	// Zero ExpiresAt → treated as "no expiry".
	noExpiry := NewBuilder().
		SetAreaDoorId(10_001_001).
		SetTownDoorId(10_001_002).
		SetOwnerCharacterId(2001).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(_map.Id(104000000)).
		SetSlot(0).
		SetDeployTime(now.Add(-10 * time.Minute)).
		// deliberately omit SetExpiresAt → zero value
		Build()

	if err := testRegistry.Put(ctx, ten, noExpiry); err != nil {
		t.Fatalf("Put: %v", err)
	}

	fake := &fakeExpiryProcessor{}
	task := &ExpiryTask{
		l:        logrus.New(),
		ctx:      ctx,
		interval: 30 * time.Second,
		newProcessor: func(l logrus.FieldLogger, tctx context.Context) expiryProcessor {
			return fake
		},
	}
	task.Run()

	if len(fake.removed) != 0 {
		t.Fatalf("expected no removal for zero ExpiresAt, got %v", fake.removed)
	}
}
