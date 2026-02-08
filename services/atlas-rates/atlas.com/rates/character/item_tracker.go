package character

import (
	"atlas-rates/bonusexp"
	"atlas-rates/data/cash"
	"atlas-rates/data/equipment"
	"atlas-rates/rate"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// ItemType represents the type of rate-affecting item
type ItemType int

const (
	ItemTypeBonusExp ItemType = iota // Equipment with bonusExp tiers
	ItemTypeCoupon                   // Cash coupons with rate/time
)

// TrackedItem represents an item being tracked for time-based rate calculations
type TrackedItem struct {
	TemplateId    uint32
	ItemType      ItemType
	RateType      rate.Type                // What rate this item affects
	BonusExpTiers []equipment.BonusExpTier // Tiers for bonusExp items

	// For bonusExp items: When the item was equipped
	EquippedSince *time.Time

	// For coupons: Acquisition info
	AcquiredAt   time.Time         // When the coupon was acquired
	BaseRate     float64           // Base rate multiplier
	DurationMins int32             // Active duration in minutes (0 = permanent)
	TimeWindows  []cash.TimeWindow // Active time windows (empty = always active)
}

// IsExpired returns true if a coupon has expired
func (t TrackedItem) IsExpired() bool {
	if t.ItemType != ItemTypeCoupon || t.DurationMins <= 0 {
		return false
	}
	expiresAt := t.AcquiredAt.Add(time.Duration(t.DurationMins) * time.Minute)
	return time.Now().After(expiresAt)
}

// GetCouponMultiplier returns the multiplier for a coupon item
// For bonusExp items, use ComputeBonusExpMultiplier instead
func (t TrackedItem) GetCouponMultiplier() float64 {
	return t.GetCouponMultiplierAt(time.Now(), false)
}

// GetCouponMultiplierAt returns the multiplier for a coupon item at a specific time
// The isHoliday parameter should be true if the given time is a holiday
func (t TrackedItem) GetCouponMultiplierAt(checkTime time.Time, isHoliday bool) float64 {
	if t.ItemType != ItemTypeCoupon {
		return 1.0
	}
	if t.IsExpired() {
		return 1.0
	}
	if !t.IsActiveAt(checkTime, isHoliday) {
		return 1.0
	}
	return t.BaseRate
}

// IsActiveAt checks if the coupon is active at a given time based on time windows
// If no time windows are defined, returns true (always active)
func (t TrackedItem) IsActiveAt(checkTime time.Time, isHoliday bool) bool {
	if len(t.TimeWindows) == 0 {
		return true
	}

	hour := checkTime.Hour()
	weekday := checkTime.Weekday()

	// Map Go's time.Weekday to WZ day abbreviations
	dayAbbreviations := map[time.Weekday]string{
		time.Monday:    "MON",
		time.Tuesday:   "TUE",
		time.Wednesday: "WED",
		time.Thursday:  "THU",
		time.Friday:    "FRI",
		time.Saturday:  "SAT",
		time.Sunday:    "SUN",
	}
	dayAbbr := dayAbbreviations[weekday]

	for _, tw := range t.TimeWindows {
		// Check holiday window
		if isHoliday && tw.Day == "HOL" {
			if hour >= tw.StartHour && hour < tw.EndHour {
				return true
			}
			// EndHour of 24 means through midnight
			if tw.EndHour == 24 && hour >= tw.StartHour {
				return true
			}
		}

		// Check regular day window
		if tw.Day == dayAbbr {
			if hour >= tw.StartHour && hour < tw.EndHour {
				return true
			}
			// EndHour of 24 means through midnight
			if tw.EndHour == 24 && hour >= tw.StartHour {
				return true
			}
		}
	}

	return false
}

// ItemTracker tracks time-based rate items per character
type ItemTracker struct {
	lock  sync.RWMutex
	items map[tenant.Model]map[uint32]map[uint32]TrackedItem // tenant -> characterId -> templateId -> TrackedItem
}

var itemTracker *ItemTracker
var itemTrackerOnce sync.Once

// GetItemTracker returns the singleton item tracker instance
func GetItemTracker() *ItemTracker {
	itemTrackerOnce.Do(func() {
		itemTracker = &ItemTracker{
			items: make(map[tenant.Model]map[uint32]map[uint32]TrackedItem),
		}
	})
	return itemTracker
}

// TrackItem starts tracking a time-based rate item
func (t *ItemTracker) TrackItem(tenant tenant.Model, characterId uint32, item TrackedItem) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, ok := t.items[tenant]; !ok {
		t.items[tenant] = make(map[uint32]map[uint32]TrackedItem)
	}
	if _, ok := t.items[tenant][characterId]; !ok {
		t.items[tenant][characterId] = make(map[uint32]TrackedItem)
	}

	t.items[tenant][characterId][item.TemplateId] = item
}

// UntrackItem stops tracking an item
func (t *ItemTracker) UntrackItem(tenant tenant.Model, characterId uint32, templateId uint32) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if charItems, ok := t.items[tenant]; ok {
		if items, ok := charItems[characterId]; ok {
			delete(items, templateId)
		}
	}
}

// UpdateEquippedSince updates the equippedSince timestamp for a bonusExp item
func (t *ItemTracker) UpdateEquippedSince(tenant tenant.Model, characterId uint32, templateId uint32, equippedSince *time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if charItems, ok := t.items[tenant]; ok {
		if items, ok := charItems[characterId]; ok {
			if item, ok := items[templateId]; ok {
				item.EquippedSince = equippedSince
				items[templateId] = item
			}
		}
	}
}

// GetTrackedItem returns a tracked item if it exists
func (t *ItemTracker) GetTrackedItem(tenant tenant.Model, characterId uint32, templateId uint32) (TrackedItem, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if charItems, ok := t.items[tenant]; ok {
		if items, ok := charItems[characterId]; ok {
			if item, ok := items[templateId]; ok {
				return item, true
			}
		}
	}
	return TrackedItem{}, false
}

// GetAllTrackedItems returns all tracked items for a character
func (t *ItemTracker) GetAllTrackedItems(tenant tenant.Model, characterId uint32) []TrackedItem {
	t.lock.RLock()
	defer t.lock.RUnlock()

	result := make([]TrackedItem, 0)
	if charItems, ok := t.items[tenant]; ok {
		if items, ok := charItems[characterId]; ok {
			for _, item := range items {
				result = append(result, item)
			}
		}
	}
	return result
}

// ComputeItemRateFactors calculates current rate factors from all tracked items
// For bonusExp items, this uses session history to compute actual equipped time
func (t *ItemTracker) ComputeItemRateFactors(l logrus.FieldLogger, ctx context.Context, ten tenant.Model, characterId uint32) []rate.Factor {
	t.lock.RLock()
	defer t.lock.RUnlock()

	factors := make([]rate.Factor, 0)

	if charItems, ok := t.items[ten]; ok {
		if items, ok := charItems[characterId]; ok {
			for templateId, item := range items {
				var multiplier float64

				switch item.ItemType {
				case ItemTypeBonusExp:
					if item.EquippedSince == nil {
						// Not currently equipped, no bonus
						continue
					}
					_, multiplier = bonusexp.ComputeCurrentTier(l, ctx, characterId, *item.EquippedSince, item.BonusExpTiers)

				case ItemTypeCoupon:
					multiplier = item.GetCouponMultiplier()

				default:
					continue
				}

				if multiplier != 1.0 {
					source := itemSource(templateId)
					factors = append(factors, rate.NewFactor(source, item.RateType, multiplier))
				}
			}
		}
	}

	return factors
}

// CleanupExpiredItems removes expired coupon items and returns removed templateIds
func (t *ItemTracker) CleanupExpiredItems(tenant tenant.Model, characterId uint32) []uint32 {
	t.lock.Lock()
	defer t.lock.Unlock()

	removed := make([]uint32, 0)

	if charItems, ok := t.items[tenant]; ok {
		if items, ok := charItems[characterId]; ok {
			for templateId, item := range items {
				if item.IsExpired() {
					delete(items, templateId)
					removed = append(removed, templateId)
				}
			}
		}
	}

	return removed
}

func itemSource(templateId uint32) string {
	return fmt.Sprintf("item:%d", templateId)
}

// ResetForTesting clears all item tracker state (testing only)
func (t *ItemTracker) ResetForTesting() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.items = make(map[tenant.Model]map[uint32]map[uint32]TrackedItem)
}
