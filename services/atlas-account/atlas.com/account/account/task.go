package account

import (
	"atlas-account/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
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
	var to int64 = 60000
	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing transition timeout task to run every %dms, timeout session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, db, interval, timeout}
}

func (t *Timeout) Run() {
	span := opentracing.StartSpan(TimeoutTask)
	defer span.Finish()

	as, err := GetInTransition(t.l, t.db)
	if err != nil {
		return
	}
	cur := time.Now()

	t.l.Debugf("Executing timeout task.")
	for _, a := range as {
		if cur.Sub(a.UpdatedAt()) > t.timeout {
			t.l.Infof("Account [%d] was stuck in transition and will be set to logged out.", a.Id())
			ten := tenant.New(a.TenantId(), "", 0, 0)
			_ = SetLoggedOut(t.db)(ten, a.Id())
		}
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
