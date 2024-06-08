package session

import (
	"atlas-login/configuration"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"time"
)

const TimeoutTask = "timeout"

type Timeout struct {
	l        logrus.FieldLogger
	interval time.Duration
	timeout  time.Duration
}

func NewTimeout(l logrus.FieldLogger, interval time.Duration) *Timeout {
	var to int64
	c, err := configuration.GetConfiguration()

	if err != nil {
		to = 3600000
	} else {
		t, err := c.FindTask(TimeoutTask)
		if err != nil {
			to = 3600000
		} else {
			to = t.Attributes.Duration
		}
	}

	timeout := time.Duration(to) * time.Millisecond
	l.Infof("Initializing timeout task to run every %dms, timeout session older than %dms", interval.Milliseconds(), timeout.Milliseconds())
	return &Timeout{l, interval, timeout}
}

func (t *Timeout) Run() {
	span := opentracing.StartSpan(TimeoutTask)
	sessions := GetRegistry().GetAll()
	cur := time.Now()

	t.l.Debugf("Executing timeout task.")
	for _, s := range sessions {
		if cur.Sub(s.LastRequest()) > t.timeout {
			t.l.Infof("Account [%d] was auto-disconnected due to inactivity.", s.AccountId())
			DestroyById(t.l, span, GetRegistry())(s.SessionId())
		}
	}
	span.Finish()
}

func (t *Timeout) SleepTime() time.Duration {
	return t.interval
}
