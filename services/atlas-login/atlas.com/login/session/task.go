package session

import (
	"atlas-login/configuration"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func NewTimeout(l logrus.FieldLogger, interval time.Duration) *Timeout {
	var to int64
	c, err := configuration.GetServiceConfig()

	if err != nil {
		to = 3600000
	} else {
		t, err := c.FindTask(TimeoutTask)
		if err != nil {
			to = 3600000
		} else {
			to = t.Duration
		}
	}

	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing timeout task to run every %dms, timeout session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, interval, timeout}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-login").Start(context.Background(), TimeoutTask)
	defer span.End()

	cur := time.Now()

	t.l.Debugf("Executing timeout task.")
	_ = tenant.ForAll(func(ten tenant.Model) error {
		tctx := tenant.WithContext(sctx, ten)
		p := NewProcessor(t.l, tctx)
		return model.ForEachSlice(p.AllInTenantProvider, func(s Model) error {
			if cur.Sub(s.LastRequest()) > t.timeout {
				t.l.Infof("Account [%d] was auto-disconnected due to inactivity.", s.AccountId())
				p.DestroyById(s.SessionId())
			}
			return nil
		})
	})
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
