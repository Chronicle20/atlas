package tasks

import (
	"atlas-buffs/berserk"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type BerserkTick struct {
	l        logrus.FieldLogger
	interval int
}

func NewBerserkTick(l logrus.FieldLogger, interval int) *BerserkTick {
	return &BerserkTick{l, interval}
}

func (r *BerserkTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(context.Background(), "berserk_tick_task")
	defer span.End()

	_ = berserk.ProcessBerserkTicks(r.l, ctx)
}

func (r *BerserkTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
