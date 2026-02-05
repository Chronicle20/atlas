package tasks

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	"atlas-maps/map/monster"
	"atlas-maps/reactor"
	"context"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const RespawnTask = "respawn_task"

type Respawn struct {
	l        logrus.FieldLogger
	interval int
}

func NewRespawn(l logrus.FieldLogger, interval int) *Respawn {
	return &Respawn{l, interval}
}

func (r *Respawn) Run() {
	r.l.Debugf("Executing spawn task.")

	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), RespawnTask)
	defer span.End()

	cp := character.NewProcessor(r.l, ctx)
	mks := cp.GetMapsWithCharacters()
	for _, mk := range mks {
		tctx := tenant.WithContext(ctx, mk.Tenant)
		transactionId := uuid.New()
		go func(mk character.MapKey) {
			_ = monster.NewProcessor(r.l, tctx).SpawnMonsters(transactionId, mk.Field)
		}(mk)
		go func(mk character.MapKey) {
			rp := reactor.NewProcessor(r.l, tctx, producer.ProviderImpl(r.l)(tctx))
			_ = rp.SpawnAndEmit(transactionId, mk.Field)
		}(mk)
	}
}

func (r *Respawn) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
