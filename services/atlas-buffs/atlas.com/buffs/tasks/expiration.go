package tasks

import (
	"atlas-buffs/character"
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"time"
)

type Respawn struct {
	l        logrus.FieldLogger
	interval int
}

func NewExpiration(l logrus.FieldLogger, interval int) *Respawn {
	return &Respawn{l, interval}
}

func (r *Respawn) Run() {
	r.l.Debugf("Executing expiration task.")

	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(context.Background(), "expiration_task")
	defer span.End()

	_ = character.ExpireBuffs(r.l, ctx)
}

func (r *Respawn) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
