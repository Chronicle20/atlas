package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"testing"
	"time"
)

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
