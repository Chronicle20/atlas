package monster

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// SelfDestructTimerTask sweeps armed self-destruct timers. It is registered
// alongside NewDropTimerTask in registerSweepTasks, so it is leader-gated when
// leader election is on; when it is not, a double fire is a no-op because
// Registry.SelfDestruct decides the transition exactly once (design D3/D6).
type SelfDestructTimerTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewSelfDestructTimerTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *SelfDestructTimerTask {
	l.Infof("Initializing self-destruct timer task to run every %dms.", interval.Milliseconds())
	return &SelfDestructTimerTask{l: l, ctx: ctx, interval: interval}
}

func (t *SelfDestructTimerTask) Run() {
	now := time.Now()
	for key, entry := range GetSelfDestructTimerRegistry().GetAll(t.ctx) {
		t.processEntry(now, key.Tenant, key.MonsterId, entry)
	}
}

func (t *SelfDestructTimerTask) processEntry(now time.Time, ten tenant.Model, uniqueId uint32, e SelfDestructTimerEntry) {
	if now.Before(e.FireAt()) {
		return
	}

	m, err := GetMonsterRegistry().GetMonster(ten, uniqueId)
	if err != nil || !m.Alive() {
		GetSelfDestructTimerRegistry().Unregister(t.ctx, ten, uniqueId)
		return
	}

	tctx := tenant.WithContext(t.ctx, ten)
	NewProcessor(t.l, tctx).SelfDestruct(uniqueId, 0, TriggerTimer)
}

func (t *SelfDestructTimerTask) SleepTime() time.Duration {
	return t.interval
}
