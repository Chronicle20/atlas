package session

import (
	"atlas-character/character"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
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

func NewTimeout(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *Timeout {
	timeout := time.Duration(5000) * time.Millisecond
	l.Infof("Initializing timeout task to run every %dms, timeout transition session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, db, interval, timeout}
}

func (t *Timeout) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-character").Start(context.Background(), TimeoutTask)
	defer span.End()

	cur := time.Now()

	t.l.Debugf("Executing timeout task.")
	cs := GetRegistry().GetAll()
	for _, m := range cs {
		tctx := tenant.WithContext(sctx, m.Tenant())
		cp := character.NewProcessor(t.l, tctx, t.db)
		cha := channel.NewModel(m.WorldId(), m.ChannelId())

		if m.State() == StateTransition && cur.Sub(m.Age()) > t.timeout {
			t.l.Debugf("Timing out record for character [%d].", m.CharacterId())
			GetRegistry().Remove(m.Tenant(), m.CharacterId())

			err := cp.Logout(uuid.New(), m.CharacterId(), cha)
			if err != nil {
				t.l.WithError(err).Errorf("Unable to logout character [%d] as a result of session being destroyed.", m.CharacterId())
			}
		}
	}

}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
