package npc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestContext() context.Context {
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), te)
}

func TestCreateNpcInField(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	_, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	ns, err := p.GetInField(f)
	if err != nil {
		t.Fatalf("GetInField returned error: %v", err)
	}
	if len(ns) != 1 {
		t.Fatalf("Expected 1 npc, got %d", len(ns))
	}
	if ns[0].NpcId() != 1104100 {
		t.Errorf("Expected NpcId 1104100, got %d", ns[0].NpcId())
	}
	if ns[0].X() != 2830 {
		t.Errorf("Expected X 2830, got %d", ns[0].X())
	}
	if ns[0].Y() != 78 {
		t.Errorf("Expected Y 78, got %d", ns[0].Y())
	}
}

func TestCreateNpcSpawnIfAbsentSuppressesWhenPresent(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("First create returned error: %v", err)
	}

	m, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78, SpawnIfAbsent: true})
	if err != nil {
		t.Fatalf("Second create returned error: %v", err)
	}
	if m.UniqueId() != 0 {
		t.Errorf("Expected suppressed create to return UniqueId 0, got %d", m.UniqueId())
	}

	ns, err := p.GetInField(f)
	if err != nil {
		t.Fatalf("GetInField returned error: %v", err)
	}
	if len(ns) != 1 {
		t.Fatalf("Expected 1 npc after suppressed create, got %d", len(ns))
	}
}

func TestCreateNpcSpawnIfAbsentIsFieldScoped(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	fa := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).SetInstance(uuid.New()).Build()
	fb := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).SetInstance(uuid.New()).Build()

	if _, err := p.Create(fa, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("Create on field A returned error: %v", err)
	}

	m, err := p.Create(fb, RestModel{NpcId: 1104100, X: 2830, Y: 78, SpawnIfAbsent: true})
	if err != nil {
		t.Fatalf("Create on field B returned error: %v", err)
	}
	if m.UniqueId() == 0 {
		t.Errorf("Expected create on field B to happen (different instance), got suppressed")
	}

	nb, err := p.GetInField(fb)
	if err != nil {
		t.Fatalf("GetInField(fb) returned error: %v", err)
	}
	if len(nb) != 1 {
		t.Fatalf("Expected 1 npc on field B, got %d", len(nb))
	}
}

func TestCreateNpcWithoutGuardStacks(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("First create returned error: %v", err)
	}
	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("Second create returned error: %v", err)
	}

	ns, err := p.GetInField(f)
	if err != nil {
		t.Fatalf("GetInField returned error: %v", err)
	}
	if len(ns) != 2 {
		t.Fatalf("Expected 2 npcs, got %d", len(ns))
	}
}
