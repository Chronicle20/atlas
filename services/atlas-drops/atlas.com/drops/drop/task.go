package drop

import (
	"atlas-drops/configuration"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"time"
)

const ExpirationTaskName = "drop_expiration_task"

type ExpirationTask struct {
	l        logrus.FieldLogger
	interval time.Duration
}

func NewExpirationTask(l logrus.FieldLogger, interval time.Duration) *ExpirationTask {
	return &ExpirationTask{l, interval}
}

func (t *ExpirationTask) Run() {
	var expire time.Duration
	c, err := configuration.GetServiceConfig()
	if err != nil {
		expire = time.Duration(3) * time.Minute
	} else {
		tc, err := c.FindTask(ExpirationTaskName)
		if err != nil {
			expire = time.Duration(3) * time.Minute
		} else {
			expire = time.Duration(tc.Duration) * time.Millisecond
		}
	}

	sctx, span := otel.GetTracerProvider().Tracer("atlas-drops").Start(context.Background(), ExpirationTaskName)
	defer span.End()

	ds := GetRegistry().GetAllDrops()
	for _, d := range ds {
		if d.Status() == StatusAvailable {
			if d.DropTime().Add(expire).Before(time.Now()) {
				tctx := tenant.WithContext(sctx, d.Tenant())
				_ = NewProcessor(t.l, tctx).ExpireAndEmit(d)
			}
		}
	}
}

func (t *ExpirationTask) SleepTime() time.Duration {
	return t.interval
}
