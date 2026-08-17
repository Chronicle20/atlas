package tasks

import (
	"atlas-maps/map/character"
	"atlas-maps/map/monster"
	"atlas-maps/reactor"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const RespawnTask = "respawn_task"

type Respawn struct {
	l          logrus.FieldLogger
	interval   int
	envContext func(context.Context) context.Context
}

func NewRespawn(l logrus.FieldLogger, interval int, envContext func(context.Context) context.Context) *Respawn {
	return &Respawn{l, interval, envContext}
}

func (r *Respawn) Run() {
	r.l.Debugf("Executing spawn task.")

	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), RespawnTask)
	defer span.End()

	cp := character.NewProcessor(r.l, ctx)
	processMapsWithCharacters(r.l, ctx, cp.GetMapsWithCharacters(), spawnMonstersAndReactors(r.l), r.envContext)
}

// spawnMonstersAndReactors spawns monsters and reactors for one tenant-scoped
// map key, each on its own goroutine. Injected into processMapsWithCharacters
// so the pure sweep logic can be tested with a spy in place of the real
// spawn/emit side effects.
func spawnMonstersAndReactors(l logrus.FieldLogger) func(ctx context.Context, transactionId uuid.UUID, mk character.MapKey) {
	return func(ctx context.Context, transactionId uuid.UUID, mk character.MapKey) {
		routine.Go(l, ctx, func(_ context.Context) {
			_ = monster.NewProcessor(l, ctx).SpawnMonsters(transactionId, mk.Field)
		})
		routine.Go(l, ctx, func(_ context.Context) {
			rp := reactor.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
			_ = rp.SpawnAndEmit(transactionId, mk.Field)
		})
	}
}

// processMapsWithCharacters originates this pod's own environment identity
// onto each map key's per-tenant context before dispatching the spawn
// goroutines -- monster spawn and reactor SpawnAndEmit both emit real Kafka
// events, so an empty ENVIRONMENT header would make decide() fail open per
// FR-1.8 and every live deployment, not just this pod's, would spawn.
func processMapsWithCharacters(l logrus.FieldLogger, ctx context.Context, mks []character.MapKey, spawn func(ctx context.Context, transactionId uuid.UUID, mk character.MapKey), envContext func(context.Context) context.Context) {
	for _, mk := range mks {
		tctx := envContext(tenant.WithContext(ctx, mk.Tenant))
		transactionId := uuid.New()
		spawn(tctx, transactionId, mk)
	}
}

func (r *Respawn) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
