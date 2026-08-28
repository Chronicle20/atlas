package summon

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// identityEnvContext is a no-op envContext for tests that don't care about
// environment origination -- it just returns ctx unchanged.
func identityEnvContext(ctx context.Context) context.Context { return ctx }

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since summon/ sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/) and must not import atlas-env even
// from a test file.
type envMarkerKey string

const envMarker = envMarkerKey("marker")

func markerEnvContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, envMarker, "stamped")
}

// TestExpiryTaskAppliesEnvContextToDespawn pins that ExpiryTask.Run threads
// envContext's result into the per-tenant context handed to newProcessor --
// the context that ultimately reaches the DESTROYED emit. Without this, the
// expiry sweep's despawn events would carry an empty ENVIRONMENT header and
// fail decide() open per FR-1.8, actioning the sweep on every live
// deployment rather than just this pod's.
func TestExpiryTaskAppliesEnvContextToDespawn(t *testing.T) {
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

	now := time.Now()
	expired := NewBuilder().SetId(3000001).SetOwnerCharacterId(42).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).
		SetExpiresAt(now.Add(-time.Minute)).Build()
	if err := GetRegistry().Put(ctx, ten, expired); err != nil {
		t.Fatal(err)
	}

	task := NewExpiryTask(logrus.New(), context.Background(), time.Second, markerEnvContext)

	var gotMarker any
	task.newProcessor = func(l logrus.FieldLogger, pctx context.Context) Processor {
		gotMarker = pctx.Value(envMarker)
		return &ProcessorImpl{
			l: l, ctx: pctx, t: tenant.MustFromContext(pctx),
			emit: func(_ topic.Token, _ model.Provider[[]kafka.Message]) error { return nil },
		}
	}
	task.Run()

	if gotMarker != "stamped" {
		t.Fatalf("envContext was not applied to the despawn context: got %v, want \"stamped\"", gotMarker)
	}
}

// TestBeholderTaskAppliesEnvContextToSweep pins that BeholderTask.Run
// threads envContext's result into the per-tenant context handed to
// sweepHeal/sweepBuff -- the context that ultimately reaches t.emit. Without
// this, the Beholder sweep's heal/buff/skill-status events would carry an
// empty ENVIRONMENT header and fail decide() open per FR-1.8.
func TestBeholderTaskAppliesEnvContextToSweep(t *testing.T) {
	ten, ctx, f := setupBeholderRegistry(t)

	now := time.Now()
	due := newBeholderModel(3000002, 42, f, now.Add(-time.Second), now.Add(-time.Second))
	if err := GetRegistry().Put(ctx, ten, due); err != nil {
		t.Fatal(err)
	}

	task := NewBeholderTask(logrus.New(), context.Background(), time.Second, markerEnvContext)
	task.pick = func(int) int { return 0 }

	var gotMarker any
	task.emit = func(emitCtx context.Context, _ topic.Token, provider model.Provider[[]kafka.Message]) error {
		if gotMarker == nil {
			gotMarker = emitCtx.Value(envMarker)
		}
		_, err := provider()
		return err
	}
	task.Run()

	if gotMarker != "stamped" {
		t.Fatalf("envContext was not applied to the sweep context: got %v, want \"stamped\"", gotMarker)
	}
}
