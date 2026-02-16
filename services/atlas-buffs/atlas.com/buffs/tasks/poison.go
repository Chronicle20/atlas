package tasks

import (
	"atlas-buffs/character"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type PoisonTick struct {
	l        logrus.FieldLogger
	interval int
}

func NewPoisonTick(l logrus.FieldLogger, interval int) *PoisonTick {
	return &PoisonTick{l, interval}
}

func (r *PoisonTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(context.Background(), "poison_tick_task")
	defer span.End()

	_ = character.ProcessPoisonTicks(r.l, ctx)
}

func (r *PoisonTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
