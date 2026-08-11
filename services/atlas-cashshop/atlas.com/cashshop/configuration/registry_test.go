package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"atlas-cashshop/configuration/tenant/cashshop"
	"atlas-cashshop/configuration/tenant/cashshop/surprise"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// An unconfigured tenant must still be able to open the stock box: the
// tenant-config fetch failing returns a zero RestModel (see GetTenantConfig),
// so the 5222000 default has to live in the accessor.
func TestGetSurpriseBoxTemplateIdsDefaultsTo5222000(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ids := GetSurpriseBoxTemplateIds(l, context.Background(), uuid.New())
	if len(ids) != 1 || ids[0] != 5222000 {
		t.Fatalf("ids = %v, want [5222000]", ids)
	}
}

func TestGetSurpriseBoxTemplateIdsUsesConfiguredList(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	tenantId := uuid.New()
	// Seed the memoized cache directly, the way the other registry tests do.
	mu.Lock()
	tenantConfig[tenantId] = tenant.RestModel{
		CashShop: cashshop.RestModel{
			Surprise: surprise.RestModel{BoxTemplateIds: []uint32{5222000, 5222002}},
		},
	}
	mu.Unlock()
	ids := GetSurpriseBoxTemplateIds(l, context.Background(), tenantId)
	if len(ids) != 2 || ids[0] != 5222000 || ids[1] != 5222002 {
		t.Fatalf("ids = %v, want [5222000 5222002]", ids)
	}
}

func TestGetCouponRateLimitDefaults(t *testing.T) {
	// A tenant whose config omits the coupons block gets the documented
	// defaults, resolved HERE — never a magic number at the call site.
	attempts, window := couponRateLimitFrom(tenant.RestModel{})
	if attempts != DefaultCouponAttempts {
		t.Errorf("attempts = %d, want %d", attempts, DefaultCouponAttempts)
	}
	if window != DefaultCouponWindow {
		t.Errorf("window = %v, want %v", window, DefaultCouponWindow)
	}
}

func TestGetCouponRateLimitFromConfig(t *testing.T) {
	cfg := tenant.RestModel{}
	cfg.CashShop.Coupons.RateLimit.Attempts = 3
	cfg.CashShop.Coupons.RateLimit.WindowSeconds = 900
	attempts, window := couponRateLimitFrom(cfg)
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if window != 15*time.Minute {
		t.Errorf("window = %v, want 15m", window)
	}
}

func TestGetCouponRateLimitRejectsZero(t *testing.T) {
	// A zero threshold would lock every account out of the coupon tab
	// permanently; a zero window would make the counter never expire.
	cfg := tenant.RestModel{}
	cfg.CashShop.Coupons.RateLimit.Attempts = 0
	cfg.CashShop.Coupons.RateLimit.WindowSeconds = 0
	attempts, window := couponRateLimitFrom(cfg)
	if attempts != DefaultCouponAttempts || window != DefaultCouponWindow {
		t.Errorf("zero config must fall back to defaults, got %d / %v", attempts, window)
	}
}
