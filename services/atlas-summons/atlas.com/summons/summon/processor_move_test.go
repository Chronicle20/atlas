package summon

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newMoveProcessor wires a ProcessorImpl backed by miniredis whose emitter
// records the topics it was handed, so Move's emit behavior is observable
// without kafka.
func newMoveProcessor(t *testing.T) (*ProcessorImpl, tenant.Model, context.Context, *[]string) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	registry = newRegistry(rc)
	idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)}

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	emitted := &[]string{}
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   ten,
		emit: func(topic string, _ model.Provider[[]kafka.Message]) error {
			*emitted = append(*emitted, topic)
			return nil
		},
	}
	return p, ten, ctx, emitted
}

// putSummon persists a summon directly via the registry so Move tests don't
// depend on Spawn/effect data.
func putSummon(t *testing.T, ctx context.Context, ten tenant.Model, ownerCharacterId uint32, x, y int16) Model {
	t.Helper()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	id := GetIdAllocator().Allocate(ctx, ten)
	m := NewBuilder().
		SetId(id).SetOwnerCharacterId(ownerCharacterId).SetSkillId(3111002).SetSkillLevel(20).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).
		SetField(f).SetX(x).SetY(y).SetHp(800).SetMaxHp(800).SetAnimated(true).Build()
	if err := GetRegistry().Put(ctx, ten, m); err != nil {
		t.Fatalf("Put: %v", err)
	}
	return m
}

func TestMoveByOwnerUpdatesPosition(t *testing.T) {
	p, ten, ctx, emitted := newMoveProcessor(t)
	m := putSummon(t, ctx, ten, 42, 100, -50)

	raw := []byte{0x01, 0x02, 0x03}
	if err := p.Move(m.Id(), 42, 250, -120, 3, raw); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}

	got, err := GetRegistry().Get(ctx, ten, m.Id())
	if err != nil {
		t.Fatal(err)
	}
	if got.X() != 250 || got.Y() != -120 {
		t.Fatalf("position not updated: got (%d,%d) want (250,-120)", got.X(), got.Y())
	}
	if got.Stance() != 3 {
		t.Fatalf("stance not updated: got %d want 3", got.Stance())
	}
	if len(*emitted) != 1 || (*emitted)[0] != EnvEventTopicSummonStatus {
		t.Fatalf("expected one MOVED emit to %s, got %v", EnvEventTopicSummonStatus, *emitted)
	}
}

func TestMoveByNonOwnerRejected(t *testing.T) {
	p, ten, ctx, emitted := newMoveProcessor(t)
	m := putSummon(t, ctx, ten, 42, 100, -50)

	if err := p.Move(m.Id(), 99, 250, -120, 3, nil); err != nil {
		t.Fatalf("Move by non-owner should be a nil no-op, got %v", err)
	}

	got, err := GetRegistry().Get(ctx, ten, m.Id())
	if err != nil {
		t.Fatal(err)
	}
	if got.X() != 100 || got.Y() != -50 {
		t.Fatalf("position changed by non-owner: got (%d,%d) want (100,-50)", got.X(), got.Y())
	}
	if len(*emitted) != 0 {
		t.Fatalf("expected no emit for non-owner move, got %v", *emitted)
	}
}
