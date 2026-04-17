package shop

import (
	"context"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const DefaultExpirationInterval = 30 * time.Second

type ExpirationTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
}

func NewExpirationTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *ExpirationTask {
	l.Infof("Initializing shop expiration task to run every %dms.", interval.Milliseconds())
	return &ExpirationTask{l: l, ctx: ctx, db: db, interval: interval}
}

func (t *ExpirationTask) Run() {
	noTenantCtx := database.WithoutTenantFilter(t.ctx)

	var results []Entity
	err := t.db.WithContext(noTenantCtx).
		Where("expires_at IS NOT NULL AND expires_at < NOW() AND state IN (?, ?)", byte(Open), byte(Maintenance)).
		Find(&results).Error
	if err != nil {
		t.l.WithError(err).Errorln("Error querying expired shops.")
		return
	}

	if len(results) == 0 {
		return
	}

	t.l.Infof("Found %d expired shops to reap.", len(results))

	for _, e := range results {
		ten, err := tenant.Create(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor)
		if err != nil {
			t.l.WithError(err).Errorf("Error creating tenant context for shop [%s].", e.Id)
			continue
		}
		tctx := tenant.WithContext(t.ctx, ten)
		p := NewProcessor(t.l, tctx, t.db)

		if err := p.CloseShopAndEmit(e.Id, e.CharacterId, CloseReasonExpired); err != nil {
			t.l.WithError(err).Errorf("Error closing expired shop [%s].", e.Id)
		}
	}
}

func (t *ExpirationTask) SleepTime() time.Duration {
	return t.interval
}
