package frederick

import (
	"context"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const DefaultCleanupInterval = 6 * time.Hour
const CleanupAge = 100 * 24 * time.Hour

type CleanupTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
}

func NewCleanupTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *CleanupTask {
	l.Infof("Initializing Frederick cleanup task to run every %dms.", interval.Milliseconds())
	return &CleanupTask{l: l, ctx: ctx, db: db, interval: interval}
}

func (t *CleanupTask) Run() {
	noTenantCtx := database.WithoutTenantFilter(t.ctx)
	cutoff := time.Now().Add(-CleanupAge)

	rows, err := cleanupExpiredItems(cutoff)(t.db.WithContext(noTenantCtx))()
	if err != nil {
		t.l.WithError(err).Errorln("Error cleaning up expired Frederick items.")
	} else if rows > 0 {
		t.l.Infof("Cleaned up %d expired Frederick items.", rows)
	}

	rows, err = cleanupExpiredMesos(cutoff)(t.db.WithContext(noTenantCtx))()
	if err != nil {
		t.l.WithError(err).Errorln("Error cleaning up expired Frederick mesos.")
	} else if rows > 0 {
		t.l.Infof("Cleaned up %d expired Frederick meso records.", rows)
	}
}

func (t *CleanupTask) SleepTime() time.Duration {
	return t.interval
}
