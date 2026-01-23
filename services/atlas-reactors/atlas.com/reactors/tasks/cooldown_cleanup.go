package tasks

import (
	"atlas-reactors/reactor"
	"time"

	"github.com/sirupsen/logrus"
)

type CooldownCleanup struct {
	l logrus.FieldLogger
}

func NewCooldownCleanup(l logrus.FieldLogger) *CooldownCleanup {
	return &CooldownCleanup{l: l}
}

func (c *CooldownCleanup) Run() {
	c.l.Debugf("Running cooldown cleanup task.")
	reactor.GetRegistry().CleanupExpiredCooldowns()
}

func (c *CooldownCleanup) SleepTime() time.Duration {
	return 60 * time.Second
}
