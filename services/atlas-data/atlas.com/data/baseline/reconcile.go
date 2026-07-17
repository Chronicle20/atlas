package baseline

import (
	"context"
	"errors"

	minio "atlas-data/storage/minio"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// restoreIntent is a tenant_baselines row still in StatusRestoring — a restore
// that began but never reached StatusComplete (pod killed / process crashed
// mid-COPY, or a cancellation that outran the detached context). region/major/
// minor identify which canonical baseline to re-run.
type restoreIntent struct {
	TenantID     string `gorm:"column:tenant_id"`
	Region       string `gorm:"column:region"`
	MajorVersion int    `gorm:"column:major_version"`
	MinorVersion int    `gorm:"column:minor_version"`
}

// pendingRestores returns every tenant whose restore is still StatusRestoring.
func pendingRestores(ctx context.Context, db *gorm.DB) ([]restoreIntent, error) {
	var rows []restoreIntent
	err := db.WithContext(ctx).
		Table("tenant_baselines").
		Select("tenant_id", "region", "major_version", "minor_version").
		Where("status = ?", StatusRestoring).
		Scan(&rows).Error
	return rows, err
}

// Reconcile heals interrupted restores at startup. An interrupted restore
// leaves a durable StatusRestoring marker (see Restore); without this an
// operator would have to notice and re-trigger it by hand — which is why
// atlas-pr-933's item search stayed broken. Each restore is spawned via
// routine.Go so startup is never blocked; Restore itself already runs under a
// detached, timeout-bounded context and is idempotent (DELETE + COPY per
// table), so re-running a partially-applied tenant is safe.
func Reconcile(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client) {
	if mc == nil {
		l.Warn("baseline reconcile skipped: minio unavailable")
		return
	}
	pending, err := pendingRestores(ctx, db)
	if err != nil {
		l.WithError(err).Warn("baseline reconcile: unable to list pending restores")
		return
	}
	if len(pending) == 0 {
		return
	}
	l.Infof("baseline reconcile: %d interrupted restore(s) to heal", len(pending))
	for _, p := range pending {
		tid, perr := uuid.Parse(p.TenantID)
		if perr != nil {
			l.WithError(perr).Warnf("baseline reconcile: bad tenant_id %q; skipping", p.TenantID)
			continue
		}
		intent := p
		r := Restorer{DB: db, MC: mc, L: l}
		routine.Go(l, ctx, func(_ context.Context) {
			l.Infof("baseline reconcile: re-restoring tenant=%s region=%s ver=%d.%d", tid, intent.Region, intent.MajorVersion, intent.MinorVersion)
			if rerr := r.Restore(ctx, intent.Region, intent.MajorVersion, intent.MinorVersion, tid); rerr != nil {
				if errors.Is(rerr, ErrRestoreInProgress) {
					// Another replica already claimed this tenant — expected under
					// a Recreate rollout where every replica reconciles at once.
					l.Infof("baseline reconcile: tenant=%s already claimed by another replica; skipping", tid)
					return
				}
				l.WithError(rerr).Warnf("baseline reconcile: re-restore failed tenant=%s", tid)
				return
			}
			l.Infof("baseline reconcile: re-restore complete tenant=%s", tid)
		})
	}
}
