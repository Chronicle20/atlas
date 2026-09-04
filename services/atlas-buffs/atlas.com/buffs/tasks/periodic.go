package tasks

import (
	"atlas-buffs/character"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// PeriodicTick drives every row of the periodic-effect table. The interval is
// the DRIVING cadence, not any row's cadence: it must be fine enough to honor
// the shortest row (POISON, 1s), and each row is emitted only when its own
// interval has elapsed (task-214 FR-2.3).
type PeriodicTick struct {
	l        logrus.FieldLogger
	interval int
}

func NewPeriodicTick(l logrus.FieldLogger, interval int) *PeriodicTick {
	return &PeriodicTick{l, interval}
}

func (r *PeriodicTick) Run(ctx context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(ctx, "periodic_tick_task")
	defer span.End()

	_ = character.ProcessPeriodicTicks(r.l, ctx)
}

func (r *PeriodicTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
