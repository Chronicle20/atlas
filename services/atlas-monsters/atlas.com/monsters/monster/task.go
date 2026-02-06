package monster

import (
	"time"

	"github.com/sirupsen/logrus"
)

type RegistryAudit struct {
	l        logrus.FieldLogger
	interval time.Duration
}

func NewRegistryAudit(l logrus.FieldLogger, interval time.Duration) *RegistryAudit {
	l.Infof("Initializing audit task to run every %dms.", interval.Milliseconds())
	return &RegistryAudit{l, interval}
}

func (t *RegistryAudit) Run() {
	mapsTracked := len(GetMonsterRegistry().mapMonsterReg)
	monsTracked := len(GetMonsterRegistry().monsterReg)
	t.l.Debugf("Registry Audit. Maps [%d]. Monsters [%d].", mapsTracked, monsTracked)
}

func (t *RegistryAudit) SleepTime() time.Duration {
	return t.interval
}
