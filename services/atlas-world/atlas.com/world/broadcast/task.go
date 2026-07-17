package broadcast

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const SweepTask = "broadcast_sweep"

// Sweep is the leader-gated task (Task 9) that expires active broadcast
// entries and promotes the next pending entry, once per tenant, every tick.
// Must only be registered on the leader-elected pod - see main.go's
// WORLD_BROADCAST_LEADER_* wiring.
type Sweep struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewSweep(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *Sweep {
	l.Infof("Initializing %s task to run every %dms.", SweepTask, interval.Milliseconds())
	return &Sweep{
		l:        l,
		ctx:      ctx,
		interval: interval,
	}
}

func (t *Sweep) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-world").Start(t.ctx, SweepTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", SweepTask)
	err := model.ForEachSlice(model.FixedProvider(GetRegistry().Tenants()), func(te tenant.Model) error {
		tctx := tenant.WithContext(sctx, te)
		return NewProcessor(t.l, tctx).SweepTenant()
	})
	if err != nil {
		t.l.WithError(err).Errorf("Encountered error when sweeping broadcast queues.")
	}
}

func (t *Sweep) SleepTime() time.Duration {
	return t.interval
}
