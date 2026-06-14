package door

import (
	"context"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// deployGrace is the minimum time between a door's deploy and when the
// expiry sweep is allowed to remove it (FR-6.3). This prevents a rapid
// cast→cancel sequence from removing the door before the client has
// acknowledged the spawn, which would crash the client.
const deployGrace = 3 * time.Second

// expiryProcessor is the minimal interface the ExpiryTask needs from a
// processor. *ProcessorImpl satisfies it; tests inject a fake.
type expiryProcessor interface {
	RemoveByOwner(ownerCharacterId uint32, reason string) error
}

// ExpiryTask is a periodic tasks.Task that sweeps expired doors across all
// tenants, honoring the deploy grace window (FR-6.3).
type ExpiryTask struct {
	l            logrus.FieldLogger
	ctx          context.Context
	interval     time.Duration
	newProcessor func(l logrus.FieldLogger, ctx context.Context) expiryProcessor
}

// NewExpiryTask wires the production processor seam.
func NewExpiryTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *ExpiryTask {
	l.Infof("Initializing door expiry task to run every %dms.", interval.Milliseconds())
	t := &ExpiryTask{
		l:        l,
		ctx:      ctx,
		interval: interval,
	}
	t.newProcessor = func(l logrus.FieldLogger, tctx context.Context) expiryProcessor {
		return NewProcessor(l, tctx)
	}
	return t
}

// SleepTime returns the task's run interval, satisfying tasks.Task.
func (t *ExpiryTask) SleepTime() time.Duration { return t.interval }

// Run iterates all doors across all tenants and removes those whose ExpiresAt
// has passed AND whose deployTime is outside the deploy grace window (FR-6.3).
// Errors per-door are logged at Warn and skip only that door — never panic.
func (t *ExpiryTask) Run() {
	all, err := GetRegistry().GetAll(t.ctx)
	if err != nil {
		t.l.WithError(err).Errorf("door expiry sweep failed")
		return
	}
	now := time.Now()
	for ten, doors := range all {
		tctx := tenant.WithContext(t.ctx, ten)
		p := t.newProcessor(t.l, tctx)
		for _, m := range doors {
			// Skip doors with no expiry configured.
			if m.ExpiresAt().IsZero() {
				continue
			}
			// Skip doors that have not yet expired.
			if now.Before(m.ExpiresAt()) {
				continue
			}
			// FR-6.3: skip doors still within the deploy grace window.
			if now.Sub(m.DeployTime()) < deployGrace {
				continue
			}
			if err := p.RemoveByOwner(m.OwnerCharacterId(), RemoveReasonExpiry); err != nil {
				t.l.WithError(err).Warnf("failed expiring door %d", m.AreaDoorId())
			}
		}
	}
}
