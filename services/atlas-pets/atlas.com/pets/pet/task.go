package pet

import (
	"atlas-pets/character"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const HungerTask = "hunger"

type Timeout struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   time.Duration
	envContext func(context.Context) context.Context
}

// NewHungerTask builds the periodic hunger sweep. envContext originates this
// pod's own environment identity onto each owner's per-tenant context before
// EvaluateHungerAndEmit runs -- EvaluateHungerAndEmit emits a real Kafka
// event, and task sits outside env-domain-guard's permitted atlas-env import
// list (main.go, kafka/, rest/, socket/), so the caller (main.go) threads
// this in as a plain function value rather than the package importing
// atlas-env itself. Without it, the hunger event would carry an empty
// ENVIRONMENT header and fail decide() open per FR-1.8: every live
// deployment, not just this pod's, would act on it.
func NewHungerTask(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *Timeout {
	l.Infof("Initializing %s task to run every %dms", HungerTask, interval.Milliseconds())
	return &Timeout{l: l, db: db, interval: interval, envContext: envContext}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-pets").Start(context.Background(), HungerTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", HungerTask)
	cids, err := character.GetLoggedIn(sctx)()
	if err != nil {
		return
	}
	for cid, mk := range cids {
		routine.Go(t.l, sctx, func(_ context.Context) {
			p := NewProcessor(t.l, t.ownerTenantContext(sctx, mk.Tenant), t.db)
			_ = p.EvaluateHungerAndEmit(cid)
		})
	}
}

// ownerTenantContext builds the per-owner context EvaluateHungerAndEmit runs
// under: the owner's tenant, then envContext to originate this pod's own
// environment identity on top. Extracted so the origination itself is
// directly testable without standing up a DB or the registry Run's other
// callers require.
func (t *Timeout) ownerTenantContext(sctx context.Context, tn tenant.Model) context.Context {
	return t.envContext(tenant.WithContext(sctx, tn))
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
