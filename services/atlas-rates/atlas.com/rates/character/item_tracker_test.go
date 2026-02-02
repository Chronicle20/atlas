package character

import (
	"atlas-rates/data/equipment"
	"atlas-rates/rate"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func createTestTenantForTracker() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func resetItemTracker() {
	GetItemTracker().ResetForTesting()
}

func TestItemTrackerSingleton(t *testing.T) {
	tracker1 := GetItemTracker()
	tracker2 := GetItemTracker()

	if tracker1 != tracker2 {
		t.Error("GetItemTracker() does not return singleton")
	}
}

func TestTrackedItem_BonusExpMultiplier_NoTiers(t *testing.T) {
	item := TrackedItem{
		ItemType:      ItemTypeBonusExp,
		AcquiredAt:   time.Now().Add(-48 * time.Hour),
		BonusExpTiers: nil,
	}

	mult := item.GetCurrentMultiplier()
	if mult != 1.0 {
		t.Errorf("GetCurrentMultiplier() = %v, want 1.0", mult)
	}
}

func TestTrackedItem_BonusExpMultiplier_SingleTier(t *testing.T) {
	tiers := []equipment.BonusExpTier{
		{IncExpR: 10, TermStart: 0}, // +10% immediately
	}

	item := TrackedItem{
		ItemType:      ItemTypeBonusExp,
		AcquiredAt:   time.Now().Add(-1 * time.Hour),
		BonusExpTiers: tiers,
	}

	// +10% = 1.10
	mult := item.GetCurrentMultiplier()
	if mult != 1.10 {
		t.Errorf("GetCurrentMultiplier() = %v, want 1.10", mult)
	}
}

func TestTrackedItem_BonusExpMultiplier_MultipleTiers(t *testing.T) {
	tiers := []equipment.BonusExpTier{
		{IncExpR: 10, TermStart: 0},  // +10% at 0 hours
		{IncExpR: 20, TermStart: 24}, // +20% at 24 hours
		{IncExpR: 30, TermStart: 72}, // +30% at 72 hours
	}

	tests := []struct {
		name        string
		hoursAgo    int
		expectedMul float64
	}{
		{"just equipped", 0, 1.10},     // 0 hours: tier 0 (+10%)
		{"12 hours", 12, 1.10},         // 12 hours: still tier 0
		{"exactly 24 hours", 24, 1.20}, // 24 hours: tier 1 (+20%)
		{"36 hours", 36, 1.20},         // 36 hours: still tier 1
		{"exactly 72 hours", 72, 1.30}, // 72 hours: tier 2 (+30%)
		{"100 hours", 100, 1.30},       // 100 hours: still tier 2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := TrackedItem{
				ItemType:      ItemTypeBonusExp,
				AcquiredAt:    time.Now().Add(-time.Duration(tt.hoursAgo) * time.Hour),
				BonusExpTiers: tiers,
			}

			mult := item.GetCurrentMultiplier()
			if mult != tt.expectedMul {
				t.Errorf("GetCurrentMultiplier() at %d hours = %v, want %v", tt.hoursAgo, mult, tt.expectedMul)
			}
		})
	}
}

func TestTrackedItem_CouponMultiplier_Active(t *testing.T) {
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-30 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 60, // 60 minutes duration
	}

	mult := item.GetCurrentMultiplier()
	if mult != 2.0 {
		t.Errorf("GetCurrentMultiplier() = %v, want 2.0", mult)
	}
}

func TestTrackedItem_CouponMultiplier_Expired(t *testing.T) {
	item := TrackedItem{
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-90 * time.Minute),
		BaseRate:     2.0,
		DurationMins: 60, // 60 minutes duration, expired
	}

	mult := item.GetCurrentMultiplier()
	if mult != 1.0 {
		t.Errorf("GetCurrentMultiplier() for expired coupon = %v, want 1.0", mult)
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

	mult := item.GetCurrentMultiplier()
	if mult != 1.5 {
		t.Errorf("GetCurrentMultiplier() for permanent coupon = %v, want 1.5", mult)
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
	resetItemTracker()
	ten := createTestTenantForTracker()

	item := TrackedItem{
		TemplateId:   1002357,
		ItemType:     ItemTypeBonusExp,
		AcquiredAt:   time.Now(),
		RateType:     rate.TypeExp,
		BonusExpTiers: []equipment.BonusExpTier{{IncExpR: 10, TermStart: 0}},
	}

	GetItemTracker().TrackItem(ten, 12345, item)

	retrieved, found := GetItemTracker().GetTrackedItem(ten, 12345, 1002357)
	if !found {
		t.Fatal("TrackItem did not store the item")
	}
	if retrieved.TemplateId != 1002357 {
		t.Errorf("TemplateId = %v, want 1002357", retrieved.TemplateId)
	}
}

func TestItemTrackerUntrackItem(t *testing.T) {
	resetItemTracker()
	ten := createTestTenantForTracker()

	item := TrackedItem{
		TemplateId: 1002357,
		ItemType:   ItemTypeBonusExp,
		AcquiredAt: time.Now(),
		RateType:   rate.TypeExp,
	}

	GetItemTracker().TrackItem(ten, 12345, item)
	GetItemTracker().UntrackItem(ten, 12345, 1002357)

	_, found := GetItemTracker().GetTrackedItem(ten, 12345, 1002357)
	if found {
		t.Error("UntrackItem did not remove the item")
	}
}

func TestItemTrackerGetAllTrackedItems(t *testing.T) {
	resetItemTracker()
	ten := createTestTenantForTracker()

	item1 := TrackedItem{TemplateId: 1001, ItemType: ItemTypeBonusExp, AcquiredAt: time.Now()}
	item2 := TrackedItem{TemplateId: 1002, ItemType: ItemTypeCoupon, AcquiredAt: time.Now()}

	GetItemTracker().TrackItem(ten, 12345, item1)
	GetItemTracker().TrackItem(ten, 12345, item2)

	items := GetItemTracker().GetAllTrackedItems(ten, 12345)
	if len(items) != 2 {
		t.Errorf("GetAllTrackedItems() returned %v items, want 2", len(items))
	}
}

func TestItemTrackerComputeItemRateFactors(t *testing.T) {
	resetItemTracker()
	ten := createTestTenantForTracker()

	// Track a bonusExp item with +10%
	item := TrackedItem{
		TemplateId:    1002357,
		ItemType:      ItemTypeBonusExp,
		AcquiredAt:    time.Now(),
		RateType:      rate.TypeExp,
		BonusExpTiers: []equipment.BonusExpTier{{IncExpR: 10, TermStart: 0}},
	}
	GetItemTracker().TrackItem(ten, 12345, item)

	factors := GetItemTracker().ComputeItemRateFactors(ten, 12345)

	if len(factors) != 1 {
		t.Fatalf("ComputeItemRateFactors() returned %v factors, want 1", len(factors))
	}

	if factors[0].RateType() != rate.TypeExp {
		t.Errorf("Factor RateType = %v, want %v", factors[0].RateType(), rate.TypeExp)
	}
	if factors[0].Multiplier() != 1.10 {
		t.Errorf("Factor Multiplier = %v, want 1.10", factors[0].Multiplier())
	}
}

func TestItemTrackerComputeItemRateFactors_Filters1x(t *testing.T) {
	resetItemTracker()
	ten := createTestTenantForTracker()

	// Track an expired coupon (multiplier will be 1.0)
	item := TrackedItem{
		TemplateId:   5210000,
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   time.Now().Add(-2 * time.Hour),
		RateType:     rate.TypeExp,
		BaseRate:     2.0,
		DurationMins: 60, // Expired
	}
	GetItemTracker().TrackItem(ten, 12345, item)

	factors := GetItemTracker().ComputeItemRateFactors(ten, 12345)

	// Should not include the 1.0 multiplier factor
	if len(factors) != 0 {
		t.Errorf("ComputeItemRateFactors() returned %v factors for 1.0 multiplier, want 0", len(factors))
	}
}

func TestItemTrackerCleanupExpiredItems(t *testing.T) {
	resetItemTracker()
	ten := createTestTenantForTracker()

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

	GetItemTracker().TrackItem(ten, 12345, expired)
	GetItemTracker().TrackItem(ten, 12345, active)
	GetItemTracker().TrackItem(ten, 12345, bonusExp)

	removed := GetItemTracker().CleanupExpiredItems(ten, 12345)

	if len(removed) != 1 {
		t.Errorf("CleanupExpiredItems() removed %v items, want 1", len(removed))
	}
	if removed[0] != 5210000 {
		t.Errorf("Removed item templateId = %v, want 5210000", removed[0])
	}

	// Verify the remaining items
	items := GetItemTracker().GetAllTrackedItems(ten, 12345)
	if len(items) != 2 {
		t.Errorf("Remaining items = %v, want 2", len(items))
	}
}

func TestItemTrackerTenantIsolation(t *testing.T) {
	resetItemTracker()

	t1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	t2, _ := tenant.Create(uuid.New(), "KMS", 1, 2)

	item1 := TrackedItem{TemplateId: 1001, ItemType: ItemTypeBonusExp, AcquiredAt: time.Now()}
	item2 := TrackedItem{TemplateId: 2002, ItemType: ItemTypeBonusExp, AcquiredAt: time.Now()}

	GetItemTracker().TrackItem(t1, 12345, item1)
	GetItemTracker().TrackItem(t2, 12345, item2)

	items1 := GetItemTracker().GetAllTrackedItems(t1, 12345)
	items2 := GetItemTracker().GetAllTrackedItems(t2, 12345)

	if len(items1) != 1 || items1[0].TemplateId != 1001 {
		t.Errorf("Tenant 1 items incorrect")
	}
	if len(items2) != 1 || items2[0].TemplateId != 2002 {
		t.Errorf("Tenant 2 items incorrect")
	}
}
