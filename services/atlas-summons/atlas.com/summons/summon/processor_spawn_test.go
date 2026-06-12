package summon

import (
	"context"
	"testing"

	"atlas-summons/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// stubEffectSource returns a fixed effect for any skill id, mimicking atlas-data
// without a live service. It satisfies the effectSource seam on ProcessorImpl.
type stubEffectSource struct {
	eff effect.Model
	err error
}

func (s stubEffectSource) GetEffect(_ uint32, _ byte) (effect.Model, error) {
	return s.eff, s.err
}

// effectWithX builds an effect whose X (puppet HP) and Duration are set.
func effectWithX(x int16, durationMs int32) effect.Model {
	e, _ := effect.Extract(effect.RestModel{X: x, Duration: durationMs})
	return e
}

// newSpawnProcessor wires a ProcessorImpl backed by miniredis and a stub effect
// source. It also initializes the package registry + id allocator against the
// same miniredis so GetRegistry()/GetIdAllocator() resolve. A no-op emitter
// avoids needing kafka.
func newSpawnProcessor(t *testing.T, eff effect.Model) (*ProcessorImpl, tenant.Model, context.Context) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	// Replace the package singletons directly (InitRegistry/InitIdAllocator use
	// sync.Once and would not re-bind across tests sharing the same process).
	registry = newRegistry(rc)
	idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)}

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	p := &ProcessorImpl{
		l:       logrus.New(),
		ctx:     ctx,
		t:       ten,
		effects: stubEffectSource{eff: eff},
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
	}
	return p, ten, ctx
}

func TestSpawnPuppetPersistsAndIndexes(t *testing.T) {
	p, ten, ctx := newSpawnProcessor(t, effectWithX(800, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	m, err := p.Spawn(f, 42, 3111002, 20, 100, -50, 0, 0)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if m.Id() == 0 {
		t.Fatalf("expected a non-zero allocated summon id")
	}
	if m.SummonType() != SummonTypePuppet {
		t.Fatalf("expected PUPPET, got %s", m.SummonType())
	}
	if m.MovementType() != MovementStationary {
		t.Fatalf("expected stationary movement, got %d", m.MovementType())
	}
	if m.Hp() != 800 {
		t.Fatalf("expected hp == effect.X (800), got %d", m.Hp())
	}

	byOwner, err := GetRegistry().GetByOwner(ctx, ten, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(byOwner) != 1 {
		t.Fatalf("expected owner index len 1, got %d", len(byOwner))
	}
	if byOwner[0].Id() != m.Id() {
		t.Fatalf("owner index id mismatch: got %d want %d", byOwner[0].Id(), m.Id())
	}
}

func TestRecastReplacesSameSkill(t *testing.T) {
	p, ten, ctx := newSpawnProcessor(t, effectWithX(800, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	first, err := p.Spawn(f, 42, 3111002, 20, 100, -50, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.Spawn(f, 42, 3111002, 20, 110, -60, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	byOwner, err := GetRegistry().GetByOwner(ctx, ten, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(byOwner) != 1 {
		t.Fatalf("expected owner index len 1 after recast, got %d", len(byOwner))
	}
	if first.Id() == second.Id() {
		t.Fatalf("expected a new id on recast; both were %d", first.Id())
	}
	if byOwner[0].Id() != second.Id() {
		t.Fatalf("owner index should hold the second summon, got %d want %d", byOwner[0].Id(), second.Id())
	}
}

func TestSpawnUnknownSkillNoOp(t *testing.T) {
	p, ten, ctx := newSpawnProcessor(t, effectWithX(800, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	m, err := p.Spawn(f, 42, 99999999, 1, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("expected nil error for unknown skill, got %v", err)
	}
	if m.Id() != 0 {
		t.Fatalf("expected empty model for unknown skill, got id %d", m.Id())
	}
	byOwner, err := GetRegistry().GetByOwner(ctx, ten, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(byOwner) != 0 {
		t.Fatalf("expected nothing persisted for unknown skill, got %d", len(byOwner))
	}
}
