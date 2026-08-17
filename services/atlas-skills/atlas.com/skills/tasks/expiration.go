package tasks

import (
	"atlas-skills/skill"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

type ExpirationTask struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   int
	envContext func(context.Context) context.Context
}

// NewExpirationTask builds the periodic cooldown-expiration sweep.
// envContext originates this pod's own environment identity (env.Self())
// onto each expired cooldown's per-tenant context; see
// skill.ExpireCooldowns for why.
func NewExpirationTask(l logrus.FieldLogger, db *gorm.DB, interval int, envContext func(context.Context) context.Context) *ExpirationTask {
	return &ExpirationTask{l, db, interval, envContext}
}

func (r *ExpirationTask) Run() {
	r.l.Debugf("Executing expiration task.")

	ctx, span := otel.GetTracerProvider().Tracer("atlas-skills").Start(context.Background(), "expiration_task")
	defer span.End()

	skill.ExpireCooldowns(r.l, ctx, r.envContext)
}

func (r *ExpirationTask) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
