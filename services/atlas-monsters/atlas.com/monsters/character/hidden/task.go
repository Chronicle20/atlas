package hidden

import (
	"context"
	"time"

	buff "atlas-monsters/character/buff"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ReconcileInterval is how often the leader pod sweeps the hidden set
// against atlas-buffs (design D4: drift repair for lost EXPIRED events).
const ReconcileInterval = 5 * time.Minute

// ReconciliationTask prunes hidden-set members whose SuperGmHide buff no
// longer exists upstream. One-way on purpose: the inverse drift (hidden in
// atlas-buffs, absent here) degrades to pre-task behavior and self-heals on
// the next APPLIED/EXPIRED event.
type ReconciliationTask struct {
	l        logrus.FieldLogger
	interval time.Duration
	registry *Registry
	buffsFn  func(t tenant.Model, characterId uint32) ([]buff.Model, error)
}

func NewReconciliationTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *ReconciliationTask {
	l.Infof("Initializing hidden-character reconciliation task to run every %dms.", interval.Milliseconds())
	t := &ReconciliationTask{l: l, interval: interval}
	t.registry = GetRegistry()
	t.buffsFn = func(ten tenant.Model, characterId uint32) ([]buff.Model, error) {
		return buff.NewProcessor(l, tenant.WithContext(ctx, ten)).GetByCharacterId(characterId)
	}
	return t
}

func (t *ReconciliationTask) Run(ctx context.Context) {
	if t.registry == nil {
		return
	}
	all := t.registry.GetAll(ctx)
	for ten, ids := range all {
		tctx := tenant.WithContext(ctx, ten)
		for _, id := range ids {
			bs, err := t.buffsFn(ten, id)
			if err != nil {
				t.l.WithError(err).Debugf("Hidden-set reconciliation: unable to fetch buffs for character [%d]; keeping entry.", id)
				continue
			}
			if !buff.HasActiveGmHide(tctx, bs) {
				// Warn: reaching here means an EXPIRED event was lost.
				t.l.Warnf("Hidden-set reconciliation: character [%d] has no active SuperGmHide buff; removing stale entry.", id)
				if err := t.registry.Remove(ctx, ten, id); err != nil {
					t.l.WithError(err).Warnf("Hidden-set reconciliation: unable to remove stale entry for character [%d].", id)
				}
			}
		}
	}
}

func (t *ReconciliationTask) SleepTime() time.Duration {
	return t.interval
}
