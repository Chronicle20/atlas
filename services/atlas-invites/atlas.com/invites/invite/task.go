package invite

import (
	invite2 "atlas-invites/kafka/message/invite"
	"atlas-invites/kafka/producer"
	"context"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const TimeoutTask = "timeout"

type Timeout struct {
	l        logrus.FieldLogger
	interval time.Duration
	timeout  time.Duration
}

func NewInviteTimeout(l logrus.FieldLogger, interval time.Duration) *Timeout {
	var to int64 = 180000
	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing invite timeout task to run every %dms, timeout invite older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, interval, timeout}
}

func (t *Timeout) Run() {
	_, span := otel.GetTracerProvider().Tracer("atlas-invites").Start(context.Background(), TimeoutTask)
	defer span.End()

	is, err := GetRegistry().GetExpired(t.timeout)
	if err != nil {
		return
	}

	t.l.Debugf("Executing timeout task.")
	for _, i := range is {
		t.l.Infof("Invite [%d] has expired. Character [%d] will no longer be able to act upon it.", i.Id(), i.TargetId())
		err = GetRegistry().Delete(i.Tenant(), i.TargetId(), i.Type(), i.OriginatorId())
		if err != nil {
			t.l.WithError(err).Errorf("Unable to expire invite [%d].", i.Id())
			return
		}

		ctx := tenant.WithContext(context.Background(), i.Tenant())
		transactionId := uuid.New()
		err = producer.ProviderImpl(t.l)(ctx)(invite2.EnvEventStatusTopic)(rejectedStatusEventProvider(i.ReferenceId(), i.WorldId(), i.Type(), i.OriginatorId(), i.TargetId(), transactionId))
		if err != nil {
			t.l.WithError(err).Errorf("Unable to produce rejection event for [%d] denying [%d] [%s] due to timeout.", i.TargetId(), i.OriginatorId(), i.Type())
		}
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
