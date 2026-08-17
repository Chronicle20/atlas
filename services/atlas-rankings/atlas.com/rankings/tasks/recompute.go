package tasks

import (
	"atlas-rankings/configuration"
	"atlas-rankings/ranking"
	"context"
	"time"

	tenantclient "atlas-rankings/tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RecomputeTask ticks every interval (60s base tick), re-enumerates tenants
// and re-reads each tenant's configured cadence on EVERY tick — never a
// boot-time snapshot — so new tenants and config changes take effect without
// a redeploy, with staleness bounded by one tick.
type RecomputeTask struct {
	l            logrus.FieldLogger
	ctx          context.Context
	interval     time.Duration
	tenants      func() ([]tenant.Model, error)
	intervalFor  func(ctx context.Context, tenantId uuid.UUID) time.Duration
	processorFor func(ctx context.Context) ranking.Processor
	envContext   func(context.Context) context.Context
}

// NewRecomputeTask builds the periodic rankings recompute sweep. envContext
// originates this pod's own environment identity onto each tenant's
// per-tick context before intervalFor (an outbound REST call through
// RootUrlFor) and processorFor's Recompute (which itself reads characters
// over REST) run -- task sits outside env-domain-guard's permitted
// atlas-env import list (main.go, kafka/, rest/, socket/), so the caller
// (main.go) threads this in as a plain function value rather than the
// package importing atlas-env itself. Without it, both the configuration
// lookup and the character read would resolve through RootUrlFor's
// legacy-baseline fallback (empty ENVIRONMENT -> RootUrl) and silently hit
// main's environment instead of this pod's.
func NewRecomputeTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *RecomputeTask {
	return &RecomputeTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
		tenants: func() ([]tenant.Model, error) {
			return tenantclient.NewProcessor(l, ctx).GetAll()
		},
		intervalFor: func(tctx context.Context, tenantId uuid.UUID) time.Duration {
			return configuration.GetRecomputeInterval(l, tctx)(tenantId)
		},
		processorFor: func(tctx context.Context) ranking.Processor {
			return ranking.NewProcessor(l, tctx, db)
		},
		envContext: envContext,
	}
}

func (t *RecomputeTask) SleepTime() time.Duration {
	return t.interval
}

func (t *RecomputeTask) Run() {
	if t.ctx.Err() != nil {
		return
	}

	ts, err := t.tenants()
	if err != nil {
		t.l.WithError(err).Warnf("Unable to enumerate tenants; skipping rankings recompute tick.")
		return
	}

	for _, ten := range ts {
		if t.ctx.Err() != nil {
			t.l.Infof("Context cancelled mid-tick; abandoning remaining tenants for this tick.")
			return
		}

		tctx := t.tenantContext(ten)
		interval := t.intervalFor(tctx, ten.Id())
		p := t.processorFor(tctx)

		now := time.Now()
		due, err := p.IsDue(interval, now)
		if err != nil {
			t.l.WithError(err).WithField("tenant", ten.Id().String()).Warnf("Unable to determine rankings cycle due-ness; skipping tenant.")
			continue
		}
		if !due {
			continue
		}
		if err := p.Recompute(now); err != nil {
			t.l.WithError(err).WithField("tenant", ten.Id().String()).Errorf("Rankings recompute failed; continuing with remaining tenants.")
			continue
		}
	}
}

// tenantContext builds the per-tenant context intervalFor and processorFor
// run under: the tenant, then envContext to originate this pod's own
// environment identity on top. Extracted so the origination itself is
// directly testable without standing up the tenant/configuration/ranking
// dependencies Run's other callers require.
func (t *RecomputeTask) tenantContext(ten tenant.Model) context.Context {
	return t.envContext(tenant.WithContext(t.ctx, ten))
}
