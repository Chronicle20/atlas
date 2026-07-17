package tasks

import (
	"context"
	"time"

	"atlas-rankings/configuration"
	"atlas-rankings/ranking"
	tenantclient "atlas-rankings/tenant"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
}

func NewRecomputeTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *RecomputeTask {
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
	}
}

func (t *RecomputeTask) SleepTime() time.Duration {
	return t.interval
}

func (t *RecomputeTask) Run() {
	ts, err := t.tenants()
	if err != nil {
		t.l.WithError(err).Warnf("Unable to enumerate tenants; skipping rankings recompute tick.")
		return
	}

	for _, ten := range ts {
		tctx := tenant.WithContext(t.ctx, ten)
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
