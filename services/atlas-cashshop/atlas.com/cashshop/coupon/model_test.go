package coupon

import (
	"errors"
	"testing"
	"time"
)

func ptrTime(t time.Time) *time.Time { return &t }
func ptrU32(v uint32) *uint32        { return &v }

func TestRedeemableAtLadderOrder(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	for _, c := range []struct {
		name    string
		build   func() Model
		wantKey string
	}{
		{
			"active, open window, uses left",
			func() Model {
				m, _ := NewBuilder("OK").SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			"",
		},
		{
			"inactive reports NOT_REGISTERED",
			func() Model {
				m, _ := NewBuilder("OFF").SetActive(false).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyNotRegistered,
		},
		{
			"before startsAt reports NOT_REGISTERED",
			func() Model {
				m, _ := NewBuilder("EARLY").SetStartsAt(ptrTime(future)).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyNotRegistered,
		},
		{
			"after expiresAt reports EXPIRED",
			func() Model {
				m, _ := NewBuilder("OLD").SetExpiresAt(ptrTime(past)).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyExpired,
		},
		{
			"exhausted reports USAGE_LIMIT",
			func() Model {
				m, _ := NewBuilder("USED").SetMaxUses(ptrU32(1)).SetRedemptionCount(1).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyUsageLimit,
		},
		{
			"inactive AND expired reports NOT_REGISTERED — inactive wins",
			func() Model {
				m, _ := NewBuilder("BOTH").SetActive(false).SetExpiresAt(ptrTime(past)).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyNotRegistered,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.build().RedeemableAt(now)
			if c.wantKey == "" {
				if err != nil {
					t.Fatalf("want redeemable, got %v", err)
				}
				return
			}
			var re *RedemptionError
			if !errors.As(err, &re) {
				t.Fatalf("want a *RedemptionError, got %v", err)
			}
			if re.Key() != c.wantKey {
				t.Errorf("key = %q, want %q", re.Key(), c.wantKey)
			}
		})
	}
}

func TestBuilderDefaultsAndNormalization(t *testing.T) {
	m, err := NewBuilder("  maple2026 ").SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.Code() != "MAPLE2026" {
		t.Errorf("code = %q, want MAPLE2026 (the builder normalizes)", m.Code())
	}
	if !m.Active() {
		t.Error("active should default true")
	}
	if m.MaxUses() != nil {
		t.Error("maxUses should default nil (unlimited)")
	}
	if m.RedemptionCount() != 0 {
		t.Error("redemptionCount should default 0")
	}
}

func TestBuilderRejectsAnEmptyOrInvalidCoupon(t *testing.T) {
	if _, err := NewBuilder("   ").SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build(); err == nil {
		t.Error("want an error for an empty code")
	}
	if _, err := NewBuilder("OK").Build(); err == nil {
		t.Error("want an error for a coupon with no rewards")
	}
	now := time.Now()
	if _, err := NewBuilder("OK").
		SetRewards(Rewards{NewCurrencyReward(1, 1)}).
		SetStartsAt(ptrTime(now.Add(time.Hour))).
		SetExpiresAt(ptrTime(now)).
		Build(); err == nil {
		t.Error("want an error when expiresAt <= startsAt")
	}
}
