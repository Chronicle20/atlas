package channel

import (
	"atlas-channel/server"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const HeartbeatTask = "heartbeat"

type Timeout struct {
	l        logrus.FieldLogger
	interval time.Duration
}

func NewHeartbeat(l logrus.FieldLogger, interval time.Duration) *Timeout {
	l.Infof("Initializing %s task to run every %dms.", HeartbeatTask, interval.Milliseconds())
	return &Timeout{
		l:        l,
		interval: interval,
	}
}

func (t *Timeout) Run(ctx context.Context) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(ctx, HeartbeatTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", HeartbeatTask)
	_ = model.ForEachSlice(model.FixedProvider(server.NewProcessor(t.l, ctx).GetAll()), func(m server.Model) error {
		tctx := tenant.WithContext(sctx, m.Tenant())
		return NewProcessor(t.l, tctx).Register(m.Channel(), m.IpAddress(), m.Port())
	})
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
