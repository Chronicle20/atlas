package summon

import (
	"context"
	"testing"
	"time"

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

func TestExpirySweepDespawnsExpired(t *testing.T) {
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
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	now := time.Now()
	expired := NewBuilder().SetId(1000001).SetOwnerCharacterId(42).SetField(f).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).
		SetExpiresAt(now.Add(-time.Minute)).Build()
	future := NewBuilder().SetId(1000002).SetOwnerCharacterId(43).SetField(f).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).
		SetExpiresAt(now.Add(time.Hour)).Build()

	if err := GetRegistry().Put(ctx, ten, expired); err != nil {
		t.Fatal(err)
	}
	if err := GetRegistry().Put(ctx, ten, future); err != nil {
		t.Fatal(err)
	}

	task := NewExpiryTask(logrus.New(), context.Background(), time.Second)
	// Substitute a processor with a no-op emitter so the sweep does not attempt
	// a real kafka publish (which would retry for ~15s against no broker).
	task.newProcessor = func(l logrus.FieldLogger, ctx context.Context) Processor {
		return &ProcessorImpl{
			l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
			emit: func(_ string, _ model.Provider[[]kafka.Message]) error { return nil },
		}
	}
	task.Run()

	if _, err := GetRegistry().Get(ctx, ten, 1000001); err == nil {
		t.Fatalf("expected expired summon 1000001 to be despawned")
	}
	if _, err := GetRegistry().Get(ctx, ten, 1000002); err != nil {
		t.Fatalf("expected future summon 1000002 to remain, got error: %v", err)
	}
}
