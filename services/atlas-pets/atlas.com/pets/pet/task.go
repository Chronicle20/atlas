package pet

import (
	"atlas-pets/character"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const HungerTask = "hunger"

type Timeout struct {
	l        logrus.FieldLogger
	db       *gorm.DB
	interval time.Duration
}

func NewHungerTask(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *Timeout {
	l.Infof("Initializing %s task to run every %dms", HungerTask, interval.Milliseconds())
	return &Timeout{l: l, db: db, interval: interval}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-pets").Start(context.Background(), HungerTask)
	defer span.End()

	t.l.Debugf("Executing %s task.", HungerTask)
	cids, err := character.GetLoggedIn(sctx)()
	if err != nil {
		return
	}
	for cid, mk := range cids {
		routine.Go(t.l, sctx, func(_ context.Context) {
			p := NewProcessor(t.l, tenant.WithContext(sctx, mk.Tenant), t.db)
			_ = p.EvaluateHungerAndEmit(cid)
		})
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
