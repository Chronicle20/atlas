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
	monsters := GetMonsterRegistry().GetMonsters()
	var mapCount, monsterCount int
	for _, mons := range monsters {
		monsterCount += len(mons)
	}
	mapCount = len(monsters)
	t.l.Debugf("Registry Audit. Tenants [%d]. Monsters [%d].", mapCount, monsterCount)
}

func (t *RegistryAudit) SleepTime() time.Duration {
	return t.interval
}
