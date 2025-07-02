package guild

import (
	"atlas-guilds/coordinator"
	"context"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
	"time"
)

const TimeoutTask = "timeout"

type Timeout struct {
	l        logrus.FieldLogger
	db       *gorm.DB
	interval time.Duration
	timeout  time.Duration
}

func NewTransitionTimeout(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *Timeout {
	var to int64 = 5000
	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing transition timeout task to run every %dms, timeout session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, db, interval, timeout}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-guilds").Start(context.Background(), TimeoutTask)
	defer span.End()

	gs, err := coordinator.GetRegistry().GetExpired(t.timeout)
	if err != nil {
		return
	}

	t.l.Debugf("Executing timeout task.")
	for _, g := range gs {
		t.l.Infof("Guild creation coordination expired for guild [%s].", g.Name())
		tctx := tenant.WithContext(sctx, g.Tenant())
		_ = NewProcessor(t.l, tctx, t.db).CreationAgreementResponseAndEmit(g.LeaderId(), false, uuid.New())
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
