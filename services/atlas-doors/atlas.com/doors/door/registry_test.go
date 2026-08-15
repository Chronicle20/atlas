package door

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	testMiniRedis *miniredis.Miniredis
	testRegistry  *Registry
)

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()
	testMiniRedis = mr

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitIdAllocator(rc)
	InitRegistry(rc)
	testRegistry = GetRegistry()

	os.Exit(m.Run())
}

func newTestTenant() (tenant.Model, context.Context) {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t, context.Background()
}

func TestRegistryRoundTripAndIndices(t *testing.T) {
	r := testRegistry
	ten, ctx := newTestTenant()

	f := field.NewBuilder(1, 2, 100000000).Build()
	m := NewBuilder().
		SetAreaDoorId(1_000_001).
		SetTownDoorId(1_000_002).
		SetOwnerCharacterId(42).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(104000000).
		SetSlot(0).
		SetTownPortalId(0x80).
		SetDeployTime(time.Unix(1000, 0)).
		SetExpiresAt(time.Unix(1120, 0)).
		Build()

	if err := r.Put(ctx, ten, m); err != nil {
		t.Fatal(err)
	}

	got, err := r.Get(ctx, ten, 1_000_001)
	if err != nil || got.OwnerCharacterId() != 42 || got.TownPortalId() != 0x80 {
		t.Fatalf("round-trip failed: %+v err=%v", got, err)
	}
	// Verify times round-trip (millisecond precision)
	if got.DeployTime().UnixMilli() != m.DeployTime().UnixMilli() {
		t.Fatalf("deployTime mismatch: want %v got %v", m.DeployTime(), got.DeployTime())
	}
	if got.ExpiresAt().UnixMilli() != m.ExpiresAt().UnixMilli() {
		t.Fatalf("expiresAt mismatch: want %v got %v", m.ExpiresAt(), got.ExpiresAt())
	}

	inField, err := r.GetInField(ctx, ten, f)
	if err != nil {
		t.Fatalf("GetInField error: %v", err)
	}
	if len(inField) != 1 {
		t.Fatalf("field index: want 1 got %d", len(inField))
	}

	byOwner, err := r.GetByOwner(ctx, ten, 42)
	if err != nil {
		t.Fatalf("GetByOwner error: %v", err)
	}
	if len(byOwner) != 1 {
		t.Fatalf("owner index: want 1 got %d", len(byOwner))
	}

	if err := r.Remove(ctx, ten, 1_000_001); err != nil {
		t.Fatal(err)
	}
	inField, _ = r.GetInField(ctx, ten, f)
	byOwner, _ = r.GetByOwner(ctx, ten, 42)
	if len(inField) != 0 || len(byOwner) != 0 {
		t.Fatalf("indices not cleared on remove: field=%d owner=%d", len(inField), len(byOwner))
	}
	inTownParty, err := r.GetInTownParty(ctx, ten, f, _map.Id(104000000), 0, 42)
	if err != nil {
		t.Fatalf("GetInTownParty after remove error: %v", err)
	}
	if len(inTownParty) != 0 {
		t.Fatalf("town index not cleared on remove: got %d doors", len(inTownParty))
	}
}

// TestSoloNonCollisionInTownPartyIndex asserts that two solo casters
// (partyId==0, different ownerCharacterId) at the same town map each get
// their own town-party key, so GetInTownParty for caster A returns only A's door.
func TestSoloNonCollisionInTownPartyIndex(t *testing.T) {
	r := testRegistry
	ten, ctx := newTestTenant()

	f := field.NewBuilder(1, 2, 100000000).Build()
	townMap := _map.Id(104000000)

	// Caster A: ownerCharacterId=100, partyId=0, areaDoorId=2_000_001
	mA := NewBuilder().
		SetAreaDoorId(2_000_001).
		SetTownDoorId(2_000_002).
		SetOwnerCharacterId(100).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(townMap).
		SetSlot(0).
		SetTownPortalId(0x81).
		SetDeployTime(time.Unix(2000, 0)).
		SetExpiresAt(time.Unix(2120, 0)).
		Build()

	// Caster B: ownerCharacterId=200, partyId=0, areaDoorId=3_000_001
	mB := NewBuilder().
		SetAreaDoorId(3_000_001).
		SetTownDoorId(3_000_002).
		SetOwnerCharacterId(200).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(townMap).
		SetSlot(0).
		SetTownPortalId(0x82).
		SetDeployTime(time.Unix(3000, 0)).
		SetExpiresAt(time.Unix(3120, 0)).
		Build()

	if err := r.Put(ctx, ten, mA); err != nil {
		t.Fatalf("Put A: %v", err)
	}
	if err := r.Put(ctx, ten, mB); err != nil {
		t.Fatalf("Put B: %v", err)
	}

	// GetInTownParty for caster A should return ONLY A's door
	doorsA, err := r.GetInTownParty(ctx, ten, f, townMap, 0, 100)
	if err != nil {
		t.Fatalf("GetInTownParty A: %v", err)
	}
	if len(doorsA) != 1 {
		t.Fatalf("expected 1 door for caster A, got %d", len(doorsA))
	}
	if doorsA[0].OwnerCharacterId() != 100 {
		t.Fatalf("expected ownerCharacterId=100, got %d", doorsA[0].OwnerCharacterId())
	}

	// GetInTownParty for caster B should return ONLY B's door
	doorsB, err := r.GetInTownParty(ctx, ten, f, townMap, 0, 200)
	if err != nil {
		t.Fatalf("GetInTownParty B: %v", err)
	}
	if len(doorsB) != 1 {
		t.Fatalf("expected 1 door for caster B, got %d", len(doorsB))
	}
	if doorsB[0].OwnerCharacterId() != 200 {
		t.Fatalf("expected ownerCharacterId=200, got %d", doorsB[0].OwnerCharacterId())
	}

	// Clean up
	_ = r.Remove(ctx, ten, 2_000_001)
	_ = r.Remove(ctx, ten, 3_000_001)
}

// TestRegistryIsTenantScoped writes identical key material (same
// areaDoorId, same field, same townMapId/party/owner) under two tenants
// against the SAME registry instance (same Redis backing) and asserts
// tenant 2 sees a miss on all four namespaces (door, door-field, door-owner,
// door-town). If the tenant segment were dropped from any of the four keys,
// tenant 2's reads would collide with tenant 1's writes and the
// corresponding sub-test would fail.
func TestRegistryIsTenantScoped(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	r := newRegistry(rc)

	ten1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ten2, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	f := field.NewBuilder(1, 2, 100000000).Build()
	townMap := _map.Id(104000000)
	m := NewBuilder().
		SetAreaDoorId(9_000_001).
		SetTownDoorId(9_000_002).
		SetOwnerCharacterId(77).
		SetPartyId(0).
		SetField(f).
		SetTownMapId(townMap).
		SetSlot(0).
		SetTownPortalId(0x90).
		SetDeployTime(time.Unix(9000, 0)).
		SetExpiresAt(time.Unix(9120, 0)).
		Build()

	if err := r.Put(ctx, ten1, m); err != nil {
		t.Fatalf("Put ten1: %v", err)
	}

	t.Run("door", func(t *testing.T) {
		if _, err := r.Get(ctx, ten2, m.AreaDoorId()); err == nil {
			t.Fatalf("tenant 2 read tenant 1's door by id")
		}
		if got, err := r.Get(ctx, ten1, m.AreaDoorId()); err != nil || got.AreaDoorId() != m.AreaDoorId() {
			t.Fatalf("tenant 1 lost its own door: %v %+v", err, got)
		}
	})

	t.Run("door-field", func(t *testing.T) {
		if inField, err := r.GetInField(ctx, ten2, f); err != nil || len(inField) != 0 {
			t.Fatalf("tenant 2 saw tenant 1's door via field index: %v %+v", err, inField)
		}
		if inField, err := r.GetInField(ctx, ten1, f); err != nil || len(inField) != 1 {
			t.Fatalf("tenant 1 lost its own field index entry: %v %+v", err, inField)
		}
	})

	t.Run("door-owner", func(t *testing.T) {
		if byOwner, err := r.GetByOwner(ctx, ten2, m.OwnerCharacterId()); err != nil || len(byOwner) != 0 {
			t.Fatalf("tenant 2 saw tenant 1's door via owner index: %v %+v", err, byOwner)
		}
		if byOwner, err := r.GetByOwner(ctx, ten1, m.OwnerCharacterId()); err != nil || len(byOwner) != 1 {
			t.Fatalf("tenant 1 lost its own owner index entry: %v %+v", err, byOwner)
		}
	})

	t.Run("door-town", func(t *testing.T) {
		if inTown, err := r.GetInTownParty(ctx, ten2, f, townMap, 0, m.OwnerCharacterId()); err != nil || len(inTown) != 0 {
			t.Fatalf("tenant 2 saw tenant 1's door via town index: %v %+v", err, inTown)
		}
		if inTown, err := r.GetInTownParty(ctx, ten1, f, townMap, 0, m.OwnerCharacterId()); err != nil || len(inTown) != 1 {
			t.Fatalf("tenant 1 lost its own town index entry: %v %+v", err, inTown)
		}
	})
}
