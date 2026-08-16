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

func (t *Timeout) Run() {
	_, span := otel.GetTracerProvider().Tracer("atlas-invites").Start(context.Background(), TimeoutTask)
	defer span.End()

	tenants := GetRegistry().GetActiveTenants()
	for _, ten := range tenants {
		ctx := t.envContext(tenant.WithContext(context.Background(), ten))
		is := GetRegistry().GetExpired(ctx, t.timeout)

		t.l.Debugf("Executing timeout task for tenant [%s].", ten.Id().String())
		for _, i := range is {
			t.l.Infof("Invite [%d] has expired. Character [%d] will no longer be able to act upon it.", i.Id(), i.TargetId())
			err := GetRegistry().Delete(ctx, i.TargetId(), i.Type(), i.OriginatorId())
			if err != nil {
				t.l.WithError(err).Errorf("Unable to expire invite [%d].", i.Id())
				continue
			}

			transactionId := uuid.New()
			err = producer.ProviderImpl(t.l)(ctx)(invite2.EnvEventStatusTopic)(rejectedStatusEventProvider(i.ReferenceId(), i.WorldId(), i.Type(), i.OriginatorId(), i.TargetId(), transactionId))
			if err != nil {
				t.l.WithError(err).Errorf("Unable to produce rejection event for [%d] denying [%d] [%s] due to timeout.", i.TargetId(), i.OriginatorId(), i.Type())
			}
		}
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
