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
