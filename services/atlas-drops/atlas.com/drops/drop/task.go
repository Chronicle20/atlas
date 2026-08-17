package drop

import (
	"atlas-drops/configuration"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const ExpirationTaskName = "drop_expiration_task"

type ExpirationTask struct {
	l          logrus.FieldLogger
	interval   time.Duration
	envContext func(context.Context) context.Context
}

// NewExpirationTask builds the periodic drop-expiration sweep. envContext
// originates this pod's own environment identity onto each expired drop's
// per-tenant context before ExpireAndEmit produces a real Kafka event --
// drop is outside env-domain-guard's permitted atlas-env import list
// (main.go, kafka/, rest/, socket/), so the caller (main.go) threads this in
// as a plain function value rather than the package importing atlas-env
// itself. Without it, the expire event would carry an empty ENVIRONMENT
// header and fail decide() open per FR-1.8: every live deployment, not just
// this pod's, would act on the expiration.
func NewExpirationTask(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *ExpirationTask {
	return &ExpirationTask{l, interval, envContext}
}

// tenantContext builds the per-tenant context each expired drop's emit runs
// under: the tenant, then envContext to originate this pod's own
// environment identity on top. Extracted so the origination itself is
// directly testable without standing up the full expiration sweep.
func (t *ExpirationTask) tenantContext(sctx context.Context, ten tenant.Model) context.Context {
	return t.envContext(tenant.WithContext(sctx, ten))
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
				tctx := t.tenantContext(sctx, d.Tenant())
				_ = NewProcessor(t.l, tctx).ExpireAndEmit(d)
			}
		}
	}
}

func (t *ExpirationTask) SleepTime() time.Duration {
	return t.interval
}
