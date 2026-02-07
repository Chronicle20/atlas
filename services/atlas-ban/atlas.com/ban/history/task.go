package history

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const RetentionDays = 90

type HistoryPurge struct {
	l        logrus.FieldLogger
	db       *gorm.DB
	interval time.Duration
}

func NewHistoryPurge(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *HistoryPurge {
	l.Infof("Initializing login history purge task to run every %dms. Retention: %d days.", interval.Milliseconds(), RetentionDays)
	return &HistoryPurge{l, db, interval}
}

// Run deletes all login history records older than RetentionDays across all tenants.
// This intentionally bypasses the processor layer and operates without tenant context,
// performing a single global sweep rather than iterating per-tenant.
func (t *HistoryPurge) Run() {
	t.l.Debugf("Executing login history purge task.")
	cutoff := time.Now().AddDate(0, 0, -RetentionDays)
	err := t.db.Where("created_at < ?", cutoff).Delete(&Entity{}).Error
	if err != nil {
		t.l.WithError(err).Errorf("Unable to purge login history.")
	}
}

func (t *HistoryPurge) SleepTime() time.Duration {
	return t.interval
}
