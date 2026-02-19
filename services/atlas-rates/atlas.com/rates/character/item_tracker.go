package character

import (
	"atlas-rates/bonusexp"
	"atlas-rates/data/cash"
	"atlas-rates/data/equipment"
	"atlas-rates/rate"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
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

func (t TrackedItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TemplateId    uint32                  `json:"templateId"`
		ItemType      ItemType                `json:"itemType"`
		RateType      rate.Type               `json:"rateType"`
		BonusExpTiers []equipment.BonusExpTier `json:"bonusExpTiers,omitempty"`
		EquippedSince *time.Time              `json:"equippedSince,omitempty"`
		AcquiredAt    time.Time               `json:"acquiredAt"`
		BaseRate      float64                 `json:"baseRate"`
		DurationMins  int32                   `json:"durationMins"`
		TimeWindows   []cash.TimeWindow       `json:"timeWindows,omitempty"`
	}{
		TemplateId:    t.TemplateId,
		ItemType:      t.ItemType,
		RateType:      t.RateType,
		BonusExpTiers: t.BonusExpTiers,
		EquippedSince: t.EquippedSince,
		AcquiredAt:    t.AcquiredAt,
		BaseRate:      t.BaseRate,
		DurationMins:  t.DurationMins,
		TimeWindows:   t.TimeWindows,
	})
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

// itemTrackerKey is a composite key for characterId:templateId
type itemTrackerKey struct {
	CharacterId uint32
	TemplateId  uint32
}

// ItemTracker tracks time-based rate items per character using Redis
type ItemTracker struct {
	items *atlas.TenantRegistry[itemTrackerKey, TrackedItem]
}

var itemTracker *ItemTracker

// GetItemTracker returns the singleton item tracker instance
func GetItemTracker() *ItemTracker {
	return itemTracker
}

// InitItemTracker initializes the item tracker with a Redis client
func InitItemTracker(client *goredis.Client) {
	itemTracker = &ItemTracker{
		items: atlas.NewTenantRegistry[itemTrackerKey, TrackedItem](client, "rates-items", func(k itemTrackerKey) string {
			return strconv.FormatUint(uint64(k.CharacterId), 10) + ":" + strconv.FormatUint(uint64(k.TemplateId), 10)
		}),
	}
}

// TrackItem starts tracking a time-based rate item
func (t *ItemTracker) TrackItem(ctx context.Context, characterId uint32, item TrackedItem) {
	ten := tenant.MustFromContext(ctx)
	key := itemTrackerKey{CharacterId: characterId, TemplateId: item.TemplateId}
	_ = t.items.Put(ctx, ten, key, item)
}

// UntrackItem stops tracking an item
func (t *ItemTracker) UntrackItem(ctx context.Context, characterId uint32, templateId uint32) {
	ten := tenant.MustFromContext(ctx)
	key := itemTrackerKey{CharacterId: characterId, TemplateId: templateId}
	_ = t.items.Remove(ctx, ten, key)
}

// UpdateEquippedSince updates the equippedSince timestamp for a bonusExp item
func (t *ItemTracker) UpdateEquippedSince(ctx context.Context, characterId uint32, templateId uint32, equippedSince *time.Time) {
	ten := tenant.MustFromContext(ctx)
	key := itemTrackerKey{CharacterId: characterId, TemplateId: templateId}
	item, err := t.items.Get(ctx, ten, key)
	if err != nil {
		return
	}
	item.EquippedSince = equippedSince
	_ = t.items.Put(ctx, ten, key, item)
}

// GetTrackedItem returns a tracked item if it exists
func (t *ItemTracker) GetTrackedItem(ctx context.Context, characterId uint32, templateId uint32) (TrackedItem, bool) {
	ten := tenant.MustFromContext(ctx)
	key := itemTrackerKey{CharacterId: characterId, TemplateId: templateId}
	item, err := t.items.Get(ctx, ten, key)
	if err != nil {
		return TrackedItem{}, false
	}
	return item, true
}

// GetAllTrackedItems returns all tracked items for a character
func (t *ItemTracker) GetAllTrackedItems(ctx context.Context, characterId uint32) []TrackedItem {
	ten := tenant.MustFromContext(ctx)
	c := t.items.Client()
	scanPattern := "atlas:" + t.items.Namespace() + ":" + atlas.TenantKey(ten) + ":" + strconv.FormatUint(uint64(characterId), 10) + ":*"

	result := make([]TrackedItem, 0)
	var cursor uint64
	for {
		keys, next, err := c.Scan(ctx, cursor, scanPattern, 100).Result()
		if err != nil {
			break
		}
		for _, k := range keys {
			data, err := c.Get(ctx, k).Bytes()
			if err != nil {
				continue
			}
			var item TrackedItem
			if err := json.Unmarshal(data, &item); err != nil {
				continue
			}
			result = append(result, item)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result
}

// ComputeItemRateFactors calculates current rate factors from all tracked items
// For bonusExp items, this uses session history to compute actual equipped time
func (t *ItemTracker) ComputeItemRateFactors(l logrus.FieldLogger, ctx context.Context, characterId uint32) []rate.Factor {
	items := t.GetAllTrackedItems(ctx, characterId)
	factors := make([]rate.Factor, 0)

	for _, item := range items {
		var multiplier float64

		switch item.ItemType {
		case ItemTypeBonusExp:
			if item.EquippedSince == nil {
				continue
			}
			_, multiplier = bonusexp.ComputeCurrentTier(l, ctx, characterId, *item.EquippedSince, item.BonusExpTiers)

		case ItemTypeCoupon:
			multiplier = item.GetCouponMultiplier()

		default:
			continue
		}

		if multiplier != 1.0 {
			source := itemSource(item.TemplateId)
			factors = append(factors, rate.NewFactor(source, item.RateType, multiplier))
		}
	}

	return factors
}

// CleanupExpiredItems removes expired coupon items and returns removed templateIds
func (t *ItemTracker) CleanupExpiredItems(ctx context.Context, characterId uint32) []uint32 {
	items := t.GetAllTrackedItems(ctx, characterId)
	removed := make([]uint32, 0)

	for _, item := range items {
		if item.IsExpired() {
			t.UntrackItem(ctx, characterId, item.TemplateId)
			removed = append(removed, item.TemplateId)
		}
	}

	return removed
}

func itemSource(templateId uint32) string {
	return fmt.Sprintf("item:%d", templateId)
}

