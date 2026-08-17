package summon

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestRegistry(t *testing.T) (*Registry, tenant.Model, context.Context) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	reg := newRegistry(rc) // unexported constructor used by InitRegistry
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return reg, ten, tenant.WithContext(context.Background(), ten)
}

func TestRegistryPutIndexesByFieldAndOwner(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder().SetId(1000001).SetOwnerCharacterId(42).SetField(f).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).Build()

	if err := reg.Put(ctx, ten, m); err != nil {
		t.Fatal(err)
	}
	inField, err := reg.GetInField(ctx, ten, f)
	if err != nil || len(inField) != 1 || inField[0].Id() != 1000001 {
		t.Fatalf("field index miss: %v %+v", err, inField)
	}
	byOwner, err := reg.GetByOwner(ctx, ten, 42)
	if err != nil || len(byOwner) != 1 {
		t.Fatalf("owner index miss: %v %+v", err, byOwner)
	}

	if err := reg.Remove(ctx, ten, 1000001); err != nil {
		t.Fatal(err)
	}
	inField, _ = reg.GetInField(ctx, ten, f)
	byOwner, _ = reg.GetByOwner(ctx, ten, 42)
	if len(inField) != 0 || len(byOwner) != 0 {
		t.Fatalf("indexes not cleared on remove")
	}
}

// TestRegistryIsTenantScoped writes under tenant 1 and reads under tenant 2
// against the SAME registry instance (same Redis backing), using identical
// key material (same summon id, same field coordinates, same owner id) for
// both tenants. If the tenant segment were dropped from the key, tenant 2's
// reads would collide with tenant 1's writes and this test would fail.
func TestRegistryIsTenantScoped(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	reg := newRegistry(rc)

	ten1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	// ten2 differs from ten1 in region AND version, not just UUID — this pins
	// the tenant-scoped key SHAPE (TenantKey embeds region/version), not
	// merely "two distinct tenants stay separate".
	ten2, err := tenant.Create(uuid.New(), "JMS", 62, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder().SetId(1000001).SetOwnerCharacterId(42).SetField(f).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).Build()

	if err := reg.Put(ctx, ten1, m); err != nil {
		t.Fatal(err)
	}

	if _, err := reg.Get(ctx, ten2, m.Id()); err == nil {
		t.Fatalf("tenant 2 read tenant 1's summon by id")
	}
	if inField, err := reg.GetInField(ctx, ten2, f); err != nil || len(inField) != 0 {
		t.Fatalf("tenant 2 saw tenant 1's summon via field index: %v %+v", err, inField)
	}
	if byOwner, err := reg.GetByOwner(ctx, ten2, m.OwnerCharacterId()); err != nil || len(byOwner) != 0 {
		t.Fatalf("tenant 2 saw tenant 1's summon via owner index: %v %+v", err, byOwner)
	}

	// tenant 1 still sees its own data.
	if got, err := reg.Get(ctx, ten1, m.Id()); err != nil || got.Id() != m.Id() {
		t.Fatalf("tenant 1 lost its own summon: %v %+v", err, got)
	}
}
