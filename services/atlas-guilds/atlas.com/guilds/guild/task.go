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
	processExpiredCoordinations(t.l, sctx, gs, t.rejectExpiredCoordination, t.envContext)
}

// rejectExpiredCoordination rejects one expired guild-creation coordination.
// It is injected into processExpiredCoordinations so the pure sweep logic
// can be tested with a spy in place of the real processor call -- the real
// call pulls in t.db, which processExpiredCoordinations itself has no need
// to know about.
func (t *Timeout) rejectExpiredCoordination(l logrus.FieldLogger, ctx context.Context, leaderId uint32) error {
	return NewProcessor(l, ctx, t.db).CreationAgreementResponseAndEmit(leaderId, false, uuid.New())
}

// processExpiredCoordinations originates this pod's own environment identity
// onto each expired guild-creation coordination's per-tenant context before
// calling act -- coordination timeout is per-character lifecycle state driven
// by real gameplay, so an empty ENVIRONMENT header would make decide() fail
// open per FR-1.8 and every live deployment, not just this pod's, would act
// on the expired coordination. A nil envContext is a caller bug; tests
// exercise this directly since NewTransitionTimeout's own tests can't
// observe the resulting context.
func processExpiredCoordinations(l logrus.FieldLogger, ctx context.Context, gs []coordinator.Model, act func(l logrus.FieldLogger, ctx context.Context, leaderId uint32) error, envContext func(context.Context) context.Context) {
	for _, g := range gs {
		l.Infof("Guild creation coordination expired for guild [%s].", g.Name())
		tctx := envContext(tenant.WithContext(ctx, g.Tenant()))
		_ = act(l, tctx, g.LeaderId())
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
