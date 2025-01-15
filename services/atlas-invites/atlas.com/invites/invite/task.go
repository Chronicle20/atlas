package invite

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"time"
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
		_ = GetRegistry().Delete(i.Tenant(), i.TargetId(), i.Type(), i.OriginatorId())
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
