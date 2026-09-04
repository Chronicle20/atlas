package channel

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const ExpirationTask = "expire"

type Timeout struct {
	l        logrus.FieldLogger
	interval time.Duration
}

func NewExpiration(l logrus.FieldLogger, interval time.Duration) *Timeout {
	l.Infof("Initializing %s task to run every %dms.", ExpirationTask, interval.Milliseconds())
	return &Timeout{
		l:        l,
		interval: interval,
	}
}

func (t *Timeout) Run(ctx context.Context) {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-world").Start(ctx, ExpirationTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", ExpirationTask)
	// tctx below is deliberately NOT run through an environment-origination
	// DI (task-232 Step 3b): ChannelServers/RemoveByWorldAndChannel below
	// both resolve through atlas-redis's TenantRegistry, an in-memory/Redis
	// registry read+delete keyed by tenant alone -- no producer/message.Emit,
	// no outbound REST via RootUrlFor. Per the audit criterion, in-memory
	// registry reads need no environment attached.
	err := model.ForEachSlice(model.FixedProvider(GetChannelRegistry().Tenants(sctx)), func(te tenant.Model) error {
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
