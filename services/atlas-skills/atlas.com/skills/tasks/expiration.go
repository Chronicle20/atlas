package tasks

import (
	"atlas-skills/skill"
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
	"time"
)

type ExpirationTask struct {
	l        logrus.FieldLogger
	db       *gorm.DB
	interval int
}

func NewExpirationTask(l logrus.FieldLogger, db *gorm.DB, interval int) *ExpirationTask {
	return &ExpirationTask{l, db, interval}
}

func (r *ExpirationTask) Run() {
	r.l.Debugf("Executing expiration task.")

	ctx, span := otel.GetTracerProvider().Tracer("atlas-skills").Start(context.Background(), "expiration_task")
	defer span.End()

	skill.ExpireCooldowns(r.l, ctx)
}

func (r *ExpirationTask) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
