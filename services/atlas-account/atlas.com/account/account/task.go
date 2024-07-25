package account

import (
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
	var to int64 = 5000
	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing transition timeout task to run every %dms, timeout session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, db, interval, timeout}
}

func (t *Timeout) Run() {
	span := opentracing.StartSpan(TimeoutTask)
	defer span.Finish()

	as, err := GetInTransition(t.timeout)
	if err != nil {
		return
	}

	t.l.Debugf("Executing timeout task.")
	for _, a := range as {
		t.l.Infof("Account [%d] was stuck in transition and will be set to logged out.", a.AccountId)
		Get().ExpireTransition(a)
	}
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
