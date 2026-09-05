package door

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// deployGrace is the minimum time between a door's deploy and when the
// expiry sweep is allowed to remove it (FR-6.3). This prevents a rapid
// cast→cancel sequence from removing the door before the client has
// acknowledged the spawn, which would crash the client.
const deployGrace = 3 * time.Second

// expiryProcessor is the minimal interface the ExpiryTask needs from a
// processor. *ProcessorImpl satisfies it; tests inject a fake.
type expiryProcessor interface {
	RemoveByOwner(ownerCharacterId character.Id, reason string) error
}

// ExpiryTask is a periodic routine.Task that sweeps expired doors across all
// tenants, honoring the deploy grace window (FR-6.3).
type ExpiryTask struct {
	l            logrus.FieldLogger
	interval     time.Duration
	newProcessor func(l logrus.FieldLogger, ctx context.Context) expiryProcessor
	envContext   func(context.Context) context.Context
}

// NewExpiryTask wires the production processor seam. envContext originates
// this pod's own environment identity onto each tenant's per-sweep context
// before RemoveByOwner emits a real DOOR_STATUS Kafka event -- door is
// outside env-domain-guard's permitted atlas-env import list (main.go,
// kafka/, rest/, socket/), so the caller (main.go) threads this in as a
// plain function value rather than the package importing atlas-env itself.
// Without it, the REMOVED event would carry an empty ENVIRONMENT header and
// fail decide() open per FR-1.8: every live deployment, not just this pod's,
// would act on the removal.
func NewExpiryTask(l logrus.FieldLogger, interval time.Duration, envContext func(context.Context) context.Context) *ExpiryTask {
	l.Infof("Initializing door expiry task to run every %dms.", interval.Milliseconds())
	t := &ExpiryTask{
		l:          l,
		interval:   interval,
		envContext: envContext,
	}
	t.newProcessor = func(l logrus.FieldLogger, tctx context.Context) expiryProcessor {
		return NewProcessor(l, tctx)
	}
	return t
}

// tenantContext builds the per-tenant context each sweep pass runs under:
// the tenant, then envContext to originate this pod's own environment
// identity on top. Extracted so the origination itself is directly
// testable without standing up the full expiry sweep.
func (t *ExpiryTask) tenantContext(sctx context.Context, ten tenant.Model) context.Context {
	return t.envContext(tenant.WithContext(sctx, ten))
}

// SleepTime returns the task's run interval, satisfying routine.Task.
func (t *ExpiryTask) SleepTime() time.Duration { return t.interval }

// Run iterates all doors across all tenants and removes those whose ExpiresAt
// has passed AND whose deployTime is outside the deploy grace window (FR-6.3).
// Errors per-door are logged at Warn and skip only that door — never panic.
func (t *ExpiryTask) Run(ctx context.Context) {
	all, err := GetRegistry().GetAll(ctx)
	if err != nil {
		t.l.WithError(err).Errorf("door expiry sweep failed")
		return
	}
	now := time.Now()
	for ten, doors := range all {
		tctx := t.tenantContext(ctx, ten)
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
