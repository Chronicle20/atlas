package invite

import (
	invite2 "atlas-invites/kafka/message/invite"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const TimeoutTask = "timeout"

type Timeout struct {
	l          logrus.FieldLogger
	interval   time.Duration
	timeout    time.Duration
	envContext func(context.Context) context.Context
}

// NewInviteTimeout builds the expired-invite sweep. envContext originates
// this pod's own environment identity (env.Self()) onto each active
// tenant's context before the rejection event is produced -- invite is
// outside env-domain-guard's permitted atlas-env import list, so the caller
// (main.go) threads this in as a plain function value rather than the
// package importing atlas-env itself. Without it, decide() sees an empty
// ENVIRONMENT header and fails open per FR-1.8: every live deployment, not
// just this pod's, would act on the expired invite.
func NewInviteTimeout(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *Timeout {
	var to int64 = 180000
	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing invite timeout task to run every %dms, timeout invite older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, interval, timeout, envContext}
}

func (t *Timeout) Run(ctx context.Context) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-invites").Start(ctx, TimeoutTask)
	defer span.End()

	tenants := GetRegistry().GetActiveTenants()
	for _, ten := range tenants {
		tctx := tenant.WithContext(sctx, ten)
		is := GetRegistry().GetExpired(tctx, t.timeout)

		t.l.Debugf("Executing timeout task for tenant [%s].", ten.Id().String())
		processExpiredInvites(t.l, tctx, is, deleteInvite, rejectInvite, t.envContext)
	}
}

// deleteInvite removes an expired invite from the registry. It is injected
// into processExpiredInvites so the pure sweep logic can be tested with a
// spy in place of the real registry call.
func deleteInvite(l logrus.FieldLogger, ctx context.Context, i Model) error {
	return GetRegistry().Delete(ctx, i.TargetId(), i.Type(), i.OriginatorId())
}

// rejectInvite emits the rejected-status event for one expired invite. It is
// injected into processExpiredInvites so the pure sweep logic can be tested
// with a spy in place of the real Kafka producer call.
func rejectInvite(l logrus.FieldLogger, ctx context.Context, i Model) error {
	transactionId := uuid.New()
	return producer.ProviderImpl(l)(ctx)(invite2.EnvEventStatusTopic)(rejectedStatusEventProvider(i.ReferenceId(), i.WorldId(), i.Type(), i.OriginatorId(), i.TargetId(), transactionId))
}

// processExpiredInvites originates this pod's own environment identity onto
// each expired invite's context before calling del and reject -- invite
// timeout is per-character lifecycle state driven by real gameplay, so an
// empty ENVIRONMENT header would make decide() fail open per FR-1.8 and every
// live deployment, not just this pod's, would act on the expired invite. A
// nil envContext is a caller bug; tests exercise this directly since
// NewInviteTimeout's own tests can't observe the resulting context.
func processExpiredInvites(l logrus.FieldLogger, ctx context.Context, is []Model, del func(l logrus.FieldLogger, ctx context.Context, i Model) error, reject func(l logrus.FieldLogger, ctx context.Context, i Model) error, envContext func(context.Context) context.Context) {
	for _, i := range is {
		tctx := envContext(ctx)
		l.Infof("Invite [%d] has expired. Character [%d] will no longer be able to act upon it.", i.Id(), i.TargetId())
		err := del(l, tctx, i)
		if err != nil {
			l.WithError(err).Errorf("Unable to expire invite [%d].", i.Id())
			continue
		}

		err = reject(l, tctx, i)
		if err != nil {
			l.WithError(err).Errorf("Unable to produce rejection event for [%d] denying [%d] [%s] due to timeout.", i.TargetId(), i.OriginatorId(), i.Type())
		}
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
