package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Documented defaults for the coupon rate limiter, applied when a tenant has
// not configured one. Resolved here alongside the other tenant defaults so no
// call site ever carries a magic number (DOM-25).
const (
	DefaultCouponAttempts = 10
	DefaultCouponWindow   = time.Hour
)

var (
	mu           sync.RWMutex
	tenantConfig map[uuid.UUID]tenant.RestModel
)

func init() {
	tenantConfig = make(map[uuid.UUID]tenant.RestModel)
}

func GetTenantConfig(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (tenant.RestModel, error) {
	mu.RLock()
	if cfg, ok := tenantConfig[tenantId]; ok {
		mu.RUnlock()
		return cfg, nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	if cfg, ok := tenantConfig[tenantId]; ok {
		return cfg, nil
	}

	cfg, err := RequestForTenant(ctx, tenantId)(l, ctx)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch tenant config for %s, using defaults", tenantId.String())
		cfg = tenant.RestModel{}
	}
	tenantConfig[tenantId] = cfg
	return cfg, nil
}

func GetHourlyExpirations(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) map[uint32]uint32 {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)

	result := make(map[uint32]uint32)
	for _, he := range cfg.CashShop.Commodities.HourlyExpirations {
		result[he.TemplateId] = he.Hours
	}
	return result
}

// DefaultSurpriseBoxTemplateId is the stock Cash Shop Surprise box. It is
// the fallback, not a constant the open path compares against directly:
// GetSurpriseBoxTemplateIds is the only reader, and a tenant may override
// or extend the list.
const DefaultSurpriseBoxTemplateId = uint32(5222000)

// GetSurpriseBoxTemplateIds returns the cash item template ids that open as
// a Cash Shop Surprise box for this tenant. GetTenantConfig returns a zero
// RestModel when the fetch fails, so an empty list means "unconfigured" and
// falls back to the stock box rather than disabling the feature.
func GetSurpriseBoxTemplateIds(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []uint32 {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)
	if len(cfg.CashShop.Surprise.BoxTemplateIds) == 0 {
		return []uint32{DefaultSurpriseBoxTemplateId}
	}
	return cfg.CashShop.Surprise.BoxTemplateIds
}

// GetCouponRateLimit returns the number of failed coupon attempts an account
// may make per window, and the window itself.
func GetCouponRateLimit(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (uint32, time.Duration) {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)
	return couponRateLimitFrom(cfg)
}

func couponRateLimitFrom(cfg tenant.RestModel) (uint32, time.Duration) {
	rl := cfg.CashShop.Coupons.RateLimit
	attempts := uint32(DefaultCouponAttempts)
	window := DefaultCouponWindow
	// Zero is "unset", not "zero allowed": a 0 threshold would lock every
	// account out of the coupon tab and a 0 window would make the Redis
	// counter immortal.
	if rl.Attempts > 0 {
		attempts = rl.Attempts
	}
	if rl.WindowSeconds > 0 {
		window = time.Duration(rl.WindowSeconds) * time.Second
	}
	return attempts, window
}
