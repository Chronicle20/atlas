package ban

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ExpiredBanCleanup struct {
	l        logrus.FieldLogger
	db       *gorm.DB
	interval time.Duration
}

func NewExpiredBanCleanup(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *ExpiredBanCleanup {
	l.Infof("Initializing expired ban cleanup task to run every %dms.", interval.Milliseconds())
	return &ExpiredBanCleanup{l, db, interval}
}

// Run deletes all expired temporary bans across all tenants. This intentionally
// bypasses the processor layer and operates without tenant context, performing a
// single global sweep rather than iterating per-tenant.
func (t *ExpiredBanCleanup) Run() {
	t.l.Debugf("Executing expired ban cleanup task.")
	now := time.Now()
	err := t.db.Where("permanent = ? AND expires_at <= ?", false, now).Delete(&Entity{}).Error
	if err != nil {
		t.l.WithError(err).Errorf("Unable to cleanup expired bans.")
	}
}

func (t *ExpiredBanCleanup) SleepTime() time.Duration {
	return t.interval
}
