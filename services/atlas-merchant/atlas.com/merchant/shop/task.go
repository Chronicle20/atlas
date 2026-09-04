package shop

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const DefaultExpirationInterval = 30 * time.Second

type ExpirationTask struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   time.Duration
	envContext func(context.Context) context.Context
}

func NewExpirationTask(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *ExpirationTask {
	l.Infof("Initializing shop expiration task to run every %dms.", interval.Milliseconds())
	return &ExpirationTask{l: l, db: db, interval: interval, envContext: envContext}
}

func (t *ExpirationTask) Run(ctx context.Context) {
	noTenantCtx := database.WithoutTenantFilter(ctx)

	// Single source of truth for the expiry predicate (incl. Draft — a hired
	// merchant abandoned during setup must still be reaped at its 24h expiry);
	// run cross-tenant so one task instance sweeps every tenant.
	results, err := getExpired()(t.db.WithContext(noTenantCtx))()
	if err != nil {
		t.l.WithError(err).Errorln("Error querying expired shops.")
		return
	}

	if len(results) == 0 {
		return
	}

	t.l.Infof("Found %d expired shops to reap.", len(results))

	processExpiredShops(t.l, ctx, results, closeExpiredShop(t.l, t.db), t.envContext)
}

// closeExpiredShop closes one expired shop and emits its close event.
// Injected into processExpiredShops so the pure sweep logic can be tested
// with a spy in place of the real CloseShopAndEmit side effect.
func closeExpiredShop(l logrus.FieldLogger, db *gorm.DB) func(ctx context.Context, e Entity) error {
	return func(ctx context.Context, e Entity) error {
		return NewProcessor(l, ctx, db).CloseShopAndEmit(e.Id, e.CharacterId, CloseReasonExpired)
	}
}

// processExpiredShops originates this pod's own environment identity onto
// each expired shop's per-tenant context before the close-and-emit --
// CloseShopAndEmit emits a real Kafka event, so an empty ENVIRONMENT header
// would make decide() fail open per FR-1.8 and every live deployment, not
// just this pod's, would close the shop.
func processExpiredShops(l logrus.FieldLogger, ctx context.Context, results []Entity, closeShop func(ctx context.Context, e Entity) error, envContext func(context.Context) context.Context) {
	for _, e := range results {
		ten, err := tenant.Create(e.TenantId, e.TenantRegion, e.TenantMajor, e.TenantMinor)
		if err != nil {
			l.WithError(err).Errorf("Error creating tenant context for shop [%s].", e.Id)
			continue
		}
		tctx := envContext(tenant.WithContext(ctx, ten))

		if err := closeShop(tctx, e); err != nil {
			l.WithError(err).Errorf("Error closing expired shop [%s].", e.Id)
		}
	}
}

func (t *ExpirationTask) SleepTime() time.Duration {
	return t.interval
}
