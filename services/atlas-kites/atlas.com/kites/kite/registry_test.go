package kite

import (
	"atlas-kites/character"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func fieldWithMap(mapId _map.Id) field.Model {
	return field.NewBuilder(0, 1, mapId).SetInstance(uuid.Nil).Build()
}

func testContext(t *testing.T) (context.Context, tenant.Model) {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), tm
}

func testRegistry(t *testing.T) {
	t.Helper()
	s := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	InitRegistry(client)
	// InMapModelProvider composes the character index with kite ownership, so
	// every test that exercises Create/GetInMap needs the character registry
	// initialised too, sharing the same miniredis instance.
	character.InitRegistry(client)
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 104040000).SetInstance(uuid.Nil).Build()
}

func TestRegistryPutGetRemove(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)

	m := NewBuilder(1000000001, testField(), 42).
		SetName("Player").
		SetTemplateId(5080000).
		SetMessage("congrats!").
		SetPosition(320, -140).
		SetCreatedAt(time.Unix(0, 0).UTC()).
		Build()

	if err := getRegistry().Put(ctx, m); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, ok, err := getRegistry().Get(ctx, 42)
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("Get: kite not found after Put")
	}
	if got.Id() != 1000000001 || got.Message() != "congrats!" || got.X() != 320 || got.Y() != -140 {
		t.Errorf("Get returned %+v", got)
	}
	if got.Field().Instance() != uuid.Nil || got.Field().MapId() != 104040000 {
		t.Errorf("field did not round-trip: %+v", got.Field())
	}

	if err := getRegistry().Remove(ctx, 42); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok, err := getRegistry().Get(ctx, 42); ok || err != nil {
		t.Errorf("Get after Remove: found=%v err=%v, want found=false err=nil", ok, err)
	}
}

func TestRegistryNextIdIsMonotonic(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)

	a, err := getRegistry().NextId(ctx)
	if err != nil {
		t.Fatalf("NextId: %v", err)
	}
	b, err := getRegistry().NextId(ctx)
	if err != nil {
		t.Fatalf("NextId: %v", err)
	}
	if b <= a {
		t.Errorf("NextId not monotonic: %d then %d", a, b)
	}
}

func TestRegistryFieldLockIsExclusive(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	f := testField()

	ok, err := getRegistry().AcquireFieldLock(ctx, f)
	if err != nil || !ok {
		t.Fatalf("first AcquireFieldLock: ok=%v err=%v", ok, err)
	}
	ok, err = getRegistry().AcquireFieldLock(ctx, f)
	if err != nil {
		t.Fatalf("second AcquireFieldLock: %v", err)
	}
	if ok {
		t.Error("second AcquireFieldLock succeeded while the first was held")
	}
	if err := getRegistry().ReleaseFieldLock(ctx, f); err != nil {
		t.Fatalf("ReleaseFieldLock: %v", err)
	}
	ok, _ = getRegistry().AcquireFieldLock(ctx, f)
	if !ok {
		t.Error("AcquireFieldLock failed after release")
	}
}
