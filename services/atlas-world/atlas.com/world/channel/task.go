package channel

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const ExpirationTask = "expire"

type Timeout struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewExpiration(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *Timeout {
	l.Infof("Initializing %s task to run every %dms.", ExpirationTask, interval.Milliseconds())
	return &Timeout{
		l:        l,
		ctx:      ctx,
		interval: interval,
	}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-world").Start(t.ctx, ExpirationTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", ExpirationTask)
	err := model.ForEachSlice(model.FixedProvider(GetChannelRegistry().Tenants()), func(te tenant.Model) error {
		tctx := tenant.WithContext(sctx, te)
		return model.ForEachSlice(model.FixedProvider(GetChannelRegistry().ChannelServers(tctx)), func(c Model) error {
			if c.CreatedAt().Add(time.Second * 15).Before(time.Now()) {
				ch := channel.NewModel(c.WorldId(), c.ChannelId())
				return NewProcessor(t.l, tctx).Unregister(ch)
			}
			return nil
		})
	})
	if err != nil {
		t.l.WithError(err).Errorf("Encountered error when expiring channels.")
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
