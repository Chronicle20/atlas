package tasks

import (
	"atlas-maps/map/character"
	"atlas-maps/map/monster"
	"context"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"time"
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

	mks := character.GetMapsWithCharacters()
	for _, mk := range mks {
		tctx := tenant.WithContext(ctx, mk.Tenant)
		go monster.Spawn(r.l)(tctx)(mk.WorldId, mk.ChannelId, mk.MapId)
	}
}

func (r *Respawn) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
