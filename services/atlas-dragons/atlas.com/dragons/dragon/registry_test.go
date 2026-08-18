package dragon

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
	reg := newRegistry(rc)
	ten, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	return reg, ten, tenant.WithContext(context.Background(), ten)
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

func TestRegistryRoundTripPreservesWideCoordinates(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	m := NewBuilder(4242).SetField(testField()).SetX(70000).SetY(-70000).SetStance(3).SetJobId(2214).Build()

	if err := reg.Put(ctx, ten, m); err != nil {
		t.Fatal(err)
	}
	got, err := reg.Get(ctx, ten, 4242)
	if err != nil {
		t.Fatal(err)
	}
	if got.X() != 70000 || got.Y() != -70000 {
		t.Fatalf("coordinates must survive as int32, got %d,%d", got.X(), got.Y())
	}
	if got.JobId() != 2214 || got.Stance() != 3 || got.Field().MapId() != 100000000 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestRegistryIndexesByField(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	f := testField()
	if err := reg.Put(ctx, ten, NewBuilder(1).SetField(f).Build()); err != nil {
		t.Fatal(err)
	}
	if err := reg.Put(ctx, ten, NewBuilder(2).SetField(f).Build()); err != nil {
		t.Fatal(err)
	}
	ms, err := reg.GetInField(ctx, ten, f)
	if err != nil || len(ms) != 2 {
		t.Fatalf("field index miss: %v %+v", err, ms)
	}
}

// The owner character id IS the primary key, so a second Put for the same
// character overwrites rather than creating a second entity (FR-1.1).
func TestRegistryOneDragonPerCharacter(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	f := testField()
	_ = reg.Put(ctx, ten, NewBuilder(7).SetField(f).SetX(1).Build())
	_ = reg.Put(ctx, ten, NewBuilder(7).SetField(f).SetX(2).Build())

	ms, err := reg.GetInField(ctx, ten, f)
	if err != nil || len(ms) != 1 || ms[0].X() != 2 {
		t.Fatalf("expected exactly one dragon at x=2, got %v %+v", err, ms)
	}
}

func TestRegistryRemoveReportsWhetherItExisted(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	_ = reg.Put(ctx, ten, NewBuilder(9).SetField(testField()).Build())

	existed, err := reg.Remove(ctx, ten, 9)
	if err != nil || !existed {
		t.Fatalf("first remove must report existed=true, got %v %v", existed, err)
	}
	existed, err = reg.Remove(ctx, ten, 9)
	if err != nil || existed {
		t.Fatalf("second remove must report existed=false and no error, got %v %v", existed, err)
	}
	ms, _ := reg.GetInField(ctx, ten, testField())
	if len(ms) != 0 {
		t.Fatalf("field index must be cleaned up, got %+v", ms)
	}
}

func TestRegistryIsTenantIsolated(t *testing.T) {
	reg, tenA, ctxA := newTestRegistry(t)
	tenB, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctxB := tenant.WithContext(context.Background(), tenB)
	f := testField()

	_ = reg.Put(ctxA, tenA, NewBuilder(5).SetField(f).Build())

	ms, err := reg.GetInField(ctxB, tenB, f)
	if err != nil || len(ms) != 0 {
		t.Fatalf("tenant B must not see tenant A's dragon: %v %+v", err, ms)
	}
	if _, err := reg.Get(ctxB, tenB, 5); err == nil {
		t.Fatal("tenant B must not fetch tenant A's dragon by character id")
	}
}
