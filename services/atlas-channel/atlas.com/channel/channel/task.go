package channel

import (
	"atlas-channel/server"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"time"
)

const HeartbeatTask = "heartbeat"

type Timeout struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewHeartbeat(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *Timeout {
	l.Infof("Initializing %s task to run every %dms.", HeartbeatTask, interval.Milliseconds())
	return &Timeout{
		l:        l,
		ctx:      ctx,
		interval: interval,
	}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(t.ctx, HeartbeatTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", HeartbeatTask)
	_ = model.ForEachSlice(model.FixedProvider(server.GetAll()), func(m server.Model) error {
		tctx := tenant.WithContext(sctx, m.Tenant())
		return NewProcessor(t.l, tctx).Register(m.WorldId(), m.ChannelId(), m.IpAddress(), m.Port())
	})
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
