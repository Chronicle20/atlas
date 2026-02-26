package shop

import (
	"context"
	"sync"
	"time"

	database "github.com/Chronicle20/atlas-database"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const DefaultReaperInterval = 30 * time.Second

func StartExpirationReaper(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup, db *gorm.DB) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(DefaultReaperInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				l.Infoln("Expiration reaper shutting down.")
				return
			case <-ticker.C:
				reapExpired(l, ctx, db)
			}
		}
	}()
	l.Infoln("Expiration reaper started.")
}

func reapExpired(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	// Query across all tenants using WithoutTenantFilter.
	noTenantCtx := database.WithoutTenantFilter(ctx)

	var results []Entity
	err := db.WithContext(noTenantCtx).
		Where("expires_at IS NOT NULL AND expires_at < NOW() AND state IN (?, ?)", byte(Open), byte(Maintenance)).
		Find(&results).Error
	if err != nil {
		l.WithError(err).Errorln("Error querying expired shops.")
		return
	}

	if len(results) == 0 {
		return
	}

	l.Infof("Found %d expired shops to reap.", len(results))

	for _, e := range results {
		t, err := tenant.Create(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor)
		if err != nil {
			l.WithError(err).Errorf("Error creating tenant context for shop [%s].", e.Id)
			continue
		}
		tctx := tenant.WithContext(ctx, t)
		p := NewProcessor(l, tctx, db)

		if err := p.CloseShopAndEmit(e.Id, e.CharacterId, CloseReasonExpired); err != nil {
			l.WithError(err).Errorf("Error closing expired shop [%s].", e.Id)
		}
	}
}
