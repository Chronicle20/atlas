package character

import (
	"atlas-rates/data/cash"
	"atlas-rates/data/equipment"
	"atlas-rates/rate"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func createTestTenantForTracker() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestTrackedItem_BonusExp_NotCoupon(t *testing.T) {
	// BonusExp items should return 1.0 from GetCouponMultiplier since they're not coupons
	// BonusExp multipliers are computed via ComputeItemRateFactors using session history
	item := TrackedItem{
		ItemType:      ItemTypeBonusExp,
		BonusExpTiers: []equipment.BonusExpTier{{IncExpR: 10, TermStart: 0}},
	}

	mult := item.GetCouponMultiplier()
	if mult != 1.0 {
		t.Errorf("GetCouponMultiplier() for bonusExp item = %v, want 1.0", mult)
	}
}

func TestTrackedItem_CouponMultiplier_Active(t *testing.T) {
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-30 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 60, // 60 minutes duration
	}

	mult := item.GetCouponMultiplier()
	if mult != 2.0 {
		t.Errorf("GetCouponMultiplier() = %v, want 2.0", mult)
	}
}

func TestTrackedItem_CouponMultiplier_Expired(t *testing.T) {
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-90 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 60, // 60 minutes duration, expired
	}

	mult := item.GetCouponMultiplier()
	if mult != 1.0 {
		t.Errorf("GetCouponMultiplier() for expired coupon = %v, want 1.0", mult)
	}
}

func TestTrackedItem_CouponMultiplier_Permanent(t *testing.T) {
	// DurationMins = 0 means permanent
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-1000 * time.Hour),
		BaseRate:     1.5,
		DurationMins: 0, // Permanent
	}

	mult := item.GetCouponMultiplier()
	if mult != 1.5 {
		t.Errorf("GetCouponMultiplier() for permanent coupon = %v, want 1.5", mult)
	}
}

func TestTrackedItem_CouponMultiplier_WithTimeWindows(t *testing.T) {
	// Monday at 19:00 (within 18-20 window)
	mondayEvening := time.Date(2024, 1, 1, 19, 0, 0, 0, time.UTC) // Jan 1, 2024 was a Monday

	// Monday at 10:00 (outside 18-20 window)
	mondayMorning := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-30 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 0, // Permanent
		TimeWindows: []cash.TimeWindow{
			{Day: "MON", StartHour: 18, EndHour: 20},
		},
	}

	// Within time window - should return base rate
	mult := item.GetCouponMultiplierAt(mondayEvening, false)
	if mult != 2.0 {
		t.Errorf("GetCouponMultiplierAt() within window = %v, want 2.0", mult)
	}

	// Outside time window - should return 1.0
	mult = item.GetCouponMultiplierAt(mondayMorning, false)
	if mult != 1.0 {
		t.Errorf("GetCouponMultiplierAt() outside window = %v, want 1.0", mult)
	}
}

func TestTrackedItem_CouponMultiplier_AllDayWindow(t *testing.T) {
	// Coupon with 00-24 window (all day)
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-30 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 0, // Permanent
		TimeWindows: []cash.TimeWindow{
			{Day: "MON", StartHour: 0, EndHour: 24},
			{Day: "TUE", StartHour: 0, EndHour: 24},
			{Day: "WED", StartHour: 0, EndHour: 24},
			{Day: "THU", StartHour: 0, EndHour: 24},
			{Day: "FRI", StartHour: 0, EndHour: 24},
			{Day: "SAT", StartHour: 0, EndHour: 24},
			{Day: "SUN", StartHour: 0, EndHour: 24},
		},
	}

	// Any time on Monday should be active
	mondayMorning := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	mult := item.GetCouponMultiplierAt(mondayMorning, false)
	if mult != 2.0 {
		t.Errorf("GetCouponMultiplierAt() all-day window = %v, want 2.0", mult)
	}
}

func TestTrackedItem_CouponMultiplier_NoTimeWindows(t *testing.T) {
	// Coupon with no time windows (always active)
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-30 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 0, // Permanent
		TimeWindows:  nil,
	}

	// Any time should be active
	anyTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	mult := item.GetCouponMultiplierAt(anyTime, false)
	if mult != 2.0 {
		t.Errorf("GetCouponMultiplierAt() no time windows = %v, want 2.0", mult)
	}
}

func TestTrackedItem_IsExpired(t *testing.T) {
	tests := []struct {
		name         string
		durationMins int32
		acquiredAgo  time.Duration
		expected     bool
	}{
		{"permanent not expired", 0, 1000 * time.Hour, false},
		{"active not expired", 60, 30 * time.Minute, false},
		{"exactly at expiry", 60, 60 * time.Minute, true},
		{"past expiry", 60, 90 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := TrackedItem{
				ItemType:     ItemTypeCoupon,
				AcquiredAt:   time.Now().Add(-tt.acquiredAgo),
				DurationMins: tt.durationMins,
			}

			if got := item.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestItemTrackerTrackItem(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForTracker()
	ctx := createTestCtx(ten)

	item := TrackedItem{
		TemplateId:    1002357,
		ItemType:      ItemTypeBonusExp,
		AcquiredAt:    time.Now(),
		RateType:      rate.TypeExp,
		BonusExpTiers: []equipment.BonusExpTier{{IncExpR: 10, TermStart: 0}},
	}

	GetItemTracker().TrackItem(ctx, 12345, item)

	retrieved, found := GetItemTracker().GetTrackedItem(ctx, 12345, 1002357)
	if !found {
		t.Fatal("TrackItem did not store the item")
	}
	if retrieved.TemplateId != 1002357 {
		t.Errorf("TemplateId = %v, want 1002357", retrieved.TemplateId)
	}
}

func TestItemTrackerUntrackItem(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForTracker()
	ctx := createTestCtx(ten)

	item := TrackedItem{
		TemplateId: 1002357,
		ItemType:   ItemTypeBonusExp,
		AcquiredAt: time.Now(),
		RateType:   rate.TypeExp,
	}

	GetItemTracker().TrackItem(ctx, 12345, item)
	GetItemTracker().UntrackItem(ctx, 12345, 1002357)

	_, found := GetItemTracker().GetTrackedItem(ctx, 12345, 1002357)
	if found {
		t.Error("UntrackItem did not remove the item")
	}
}

func TestItemTrackerGetAllTrackedItems(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForTracker()
	ctx := createTestCtx(ten)

	item1 := TrackedItem{TemplateId: 1001, ItemType: ItemTypeBonusExp, AcquiredAt: time.Now()}
	item2 := TrackedItem{TemplateId: 1002, ItemType: ItemTypeCoupon, AcquiredAt: time.Now()}

	GetItemTracker().TrackItem(ctx, 12345, item1)
	GetItemTracker().TrackItem(ctx, 12345, item2)

	items := GetItemTracker().GetAllTrackedItems(ctx, 12345)
	if len(items) != 2 {
		t.Errorf("GetAllTrackedItems() returned %v items, want 2", len(items))
	}
}

func TestItemTrackerComputeItemRateFactors_Coupon(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForTracker()
	ctx := createTestCtx(ten)
	l, _ := test.NewNullLogger()

	// Track an active coupon with 2x rate
	item := TrackedItem{
		TemplateId:   5210000,
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now(),
		RateType:     rate.TypeExp,
		BaseRate:     2.0,
		DurationMins: 60, // 60 minutes, still active
	}
	GetItemTracker().TrackItem(ctx, 12345, item)

	factors := GetItemTracker().ComputeItemRateFactors(l, ctx, 12345)

	if len(factors) != 1 {
		t.Fatalf("ComputeItemRateFactors() returned %v factors, want 1", len(factors))
	}

	if factors[0].RateType() != rate.TypeExp {
		t.Errorf("Factor RateType = %v, want %v", factors[0].RateType(), rate.TypeExp)
	}
	if factors[0].Multiplier() != 2.0 {
		t.Errorf("Factor Multiplier = %v, want 2.0", factors[0].Multiplier())
	}
}

func TestItemTrackerComputeItemRateFactors_Filters1x(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForTracker()
	ctx := createTestCtx(ten)
	l, _ := test.NewNullLogger()

	// Track an expired coupon (multiplier will be 1.0)
	item := TrackedItem{
		TemplateId:   5210000,
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-2 * time.Hour),
		RateType:     rate.TypeExp,
		BaseRate:     2.0,
		DurationMins: 60, // Expired
	}
	GetItemTracker().TrackItem(ctx, 12345, item)

	factors := GetItemTracker().ComputeItemRateFactors(l, ctx, 12345)

	// Should not include the 1.0 multiplier factor
	if len(factors) != 0 {
		t.Errorf("ComputeItemRateFactors() returned %v factors for 1.0 multiplier, want 0", len(factors))
	}
}

func TestItemTrackerCleanupExpiredItems(t *testing.T) {
	setupTestRegistries(t)
	ten := createTestTenantForTracker()
	ctx := createTestCtx(ten)

	// Track an expired coupon
	expired := TrackedItem{
		TemplateId:   5210000,
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-2 * time.Hour),
		RateType:     rate.TypeExp,
		BaseRate:     2.0,
		DurationMins: 60,
	}

	// Track an active coupon
	active := TrackedItem{
		TemplateId:   5210001,
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now(),
		RateType:     rate.TypeExp,
		BaseRate:     2.0,
		DurationMins: 60,
	}

	// Track a bonusExp item (never expires)
	bonusExp := TrackedItem{
		TemplateId:    1002357,
		ItemType:      ItemTypeBonusExp,
		AcquiredAt:    time.Now(),
		RateType:      rate.TypeExp,
		BonusExpTiers: []equipment.BonusExpTier{{IncExpR: 10, TermStart: 0}},
	}

	GetItemTracker().TrackItem(ctx, 12345, expired)
	GetItemTracker().TrackItem(ctx, 12345, active)
	GetItemTracker().TrackItem(ctx, 12345, bonusExp)

	removed := GetItemTracker().CleanupExpiredItems(ctx, 12345)

	if len(removed) != 1 {
		t.Errorf("CleanupExpiredItems() removed %v items, want 1", len(removed))
	}
	if removed[0] != 5210000 {
		t.Errorf("Removed item templateId = %v, want 5210000", removed[0])
	}

	// Verify the remaining items
	items := GetItemTracker().GetAllTrackedItems(ctx, 12345)
	if len(items) != 2 {
		t.Errorf("Remaining items = %v, want 2", len(items))
	}
}

func TestItemTrackerTenantIsolation(t *testing.T) {
	setupTestRegistries(t)

	t1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	t2, _ := tenant.Create(uuid.New(), "KMS", 1, 2)
	ctx1 := createTestCtx(t1)
	ctx2 := createTestCtx(t2)

	item1 := TrackedItem{TemplateId: 1001, ItemType: ItemTypeBonusExp, AcquiredAt: time.Now()}
	item2 := TrackedItem{TemplateId: 2002, ItemType: ItemTypeBonusExp, AcquiredAt: time.Now()}

	GetItemTracker().TrackItem(ctx1, 12345, item1)
	GetItemTracker().TrackItem(ctx2, 12345, item2)

	items1 := GetItemTracker().GetAllTrackedItems(ctx1, 12345)
	items2 := GetItemTracker().GetAllTrackedItems(ctx2, 12345)

	if len(items1) != 1 || items1[0].TemplateId != 1001 {
		t.Errorf("Tenant 1 items incorrect")
	}
	if len(items2) != 1 || items2[0].TemplateId != 2002 {
		t.Errorf("Tenant 2 items incorrect")
	}
}
