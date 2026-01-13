package tasks

import (
	"atlas-buffs/character"
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"time"
)

type Expiration struct {
	l        logrus.FieldLogger
	interval int
}

func NewExpiration(l logrus.FieldLogger, interval int) *Expiration {
	return &Expiration{l, interval}
}

func (r *Expiration) Run() {
	r.l.Debugf("Executing expiration task.")

	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(context.Background(), "expiration_task")
	defer span.End()

	_ = character.ExpireBuffs(r.l, ctx)
}

func (r *Expiration) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
