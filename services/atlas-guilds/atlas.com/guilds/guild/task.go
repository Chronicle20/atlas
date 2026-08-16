package guild

import (
	"atlas-guilds/coordinator"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const TimeoutTask = "timeout"

type Timeout struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   time.Duration
	timeout    time.Duration
	envContext func(context.Context) context.Context
}

// NewTransitionTimeout builds the guild-creation-coordination expiry sweep.
// envContext originates this pod's own environment identity (env.Self())
// onto each expired guild's per-tenant context before
// CreationAgreementResponseAndEmit produces a real Kafka event -- guild is
// outside env-domain-guard's permitted atlas-env import list, so the caller
// (main.go) threads this in as a plain function value rather than the
// package importing atlas-env itself. Without it, decide() sees an empty
// ENVIRONMENT header and fails open per FR-1.8: every live deployment, not
// just this pod's, would act on the expired coordination.
func NewTransitionTimeout(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *Timeout {
	var to int64 = 5000
	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing transition timeout task to run every %dms, timeout session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, db, interval, timeout, envContext}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-guilds").Start(context.Background(), TimeoutTask)
	defer span.End()

	gs, err := coordinator.GetRegistry().GetExpiredAcrossTenants(t.timeout)
	if err != nil {
		return
	}

	t.l.Debugf("Executing timeout task.")
	for _, g := range gs {
		t.l.Infof("Guild creation coordination expired for guild [%s].", g.Name())
		tctx := t.envContext(tenant.WithContext(sctx, g.Tenant()))
		_ = NewProcessor(t.l, tctx, t.db).CreationAgreementResponseAndEmit(g.LeaderId(), false, uuid.New())
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
