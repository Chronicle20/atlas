package summon

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ExpiryTask periodically despawns summons whose duration has elapsed. It runs
// only on the leader-elected pod (registered from main.go's registerSweepTasks).
type ExpiryTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
	// newProcessor builds the tenant-scoped processor used to despawn expired
	// summons. It is a field so tests can substitute a processor with a no-op
	// emitter; production uses the real NewProcessor.
	newProcessor func(l logrus.FieldLogger, ctx context.Context) Processor
	// envContext originates this pod's own environment identity
	// (env.Self()) onto each tenant group's per-tenant context before the
	// despawn's DESTROYED event is emitted. This is a periodic multi-tenant
	// background sweep with no inbound request to inherit ENVIRONMENT from,
	// so the caller (main.go) threads this in as a plain function value
	// (withSelfEnvironment) rather than the package importing atlas-env
	// itself — summon/ is outside env-domain-guard's permitted import list.
	envContext func(context.Context) context.Context
}

func NewExpiryTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration, envContext func(context.Context) context.Context) *ExpiryTask {
	return &ExpiryTask{l: l, ctx: ctx, interval: interval, newProcessor: NewProcessor, envContext: envContext}
}

func (t *ExpiryTask) SleepTime() time.Duration { return t.interval }

// Run enumerates every stored summon grouped by tenant, and despawns those whose
// ExpiresAt is in the past. A tenant-scoped context is built per group so the
// processor's Despawn (registry removal + oid release + DESTROYED emit) operates
// in the correct tenant.
func (t *ExpiryTask) Run() {
	all, err := GetRegistry().GetAll(t.ctx)
	if err != nil {
		t.l.WithError(err).Errorf("Expiry sweep unable to enumerate summons.")
		return
	}
	now := time.Now()
	for ten, ms := range all {
		tctx := t.envContext(tenant.WithContext(t.ctx, ten))
		p := t.newProcessor(t.l, tctx)
		for _, m := range ms {
			if m.ExpiresAt().IsZero() || now.Before(m.ExpiresAt()) {
				continue
			}
			if err := p.Despawn(m.Id(), true); err != nil {
				t.l.WithError(err).Warnf("Expiry sweep failed to despawn summon [%d].", m.Id())
			}
		}
	}
}
