package broadcast

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const SweepTask = "broadcast_sweep"

// Sweep is the leader-gated task (Task 9) that expires active broadcast
// entries and promotes the next pending entry, once per tenant, every tick.
// Must only be registered on the leader-elected pod - see main.go's
// WORLD_BROADCAST_LEADER_* wiring.
type Sweep struct {
	l          logrus.FieldLogger
	interval   time.Duration
	envContext func(context.Context) context.Context
}

// NewSweep constructs the leader-gated broadcast sweep task. envContext
// attaches this pod's environment identity to each tenant's context before
// it reaches SweepTenant's Kafka emit -- the sweep's root ctx is a
// background/otel-span context with no inbound request to inherit
// ENVIRONMENT from, and broadcast/ is outside env-domain-guard's permitted
// atlas-env import list, so the environment is threaded in via DI rather
// than imported directly (task-232).
func NewSweep(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *Sweep {
	l.Infof("Initializing %s task to run every %dms.", SweepTask, interval.Milliseconds())
	return &Sweep{
		l:          l,
		interval:   interval,
		envContext: envContext,
	}
}

func (t *Sweep) Run(ctx context.Context) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-world").Start(ctx, SweepTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", SweepTask)
	err := model.ForEachSlice(model.FixedProvider(GetRegistry().Tenants()), func(te tenant.Model) error {
		tctx := t.envContext(tenant.WithContext(sctx, te))
		return NewProcessor(t.l, tctx).SweepTenant()
	})
	if err != nil {
		t.l.WithError(err).Errorf("Encountered error when sweeping broadcast queues.")
	}
}

func (t *Sweep) SleepTime() time.Duration {
	return t.interval
}
