package bonusexp

import (
	"atlas-rates/data/equipment"
	"atlas-rates/session"
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// ComputeCurrentTier calculates the current bonus exp tier for an equipped item
// Takes into account:
// - Session history (only gameplay time counts)
// - Midnight reset rule (hours reset at midnight, but tier is retained)
// - equippedSince timestamp (when the item was equipped)
func ComputeCurrentTier(l logrus.FieldLogger, ctx context.Context, characterId uint32, equippedSince time.Time, tiers []equipment.BonusExpTier) (int8, float64) {
	if len(tiers) == 0 {
		return 0, 1.0
	}

	now := time.Now()
	todayMidnight := truncateToDay(now)

	var tier int8
	var multiplier float64

	if equippedSince.Before(todayMidnight) {
		// Equipped before midnight - need to handle retained tier
		tier, multiplier = computeWithMidnightRetention(l, ctx, characterId, equippedSince, todayMidnight, now, tiers)
	} else {
		// Equipped after midnight - simple calculation
		tier, multiplier = computeSimple(l, ctx, characterId, equippedSince, now, tiers)
	}

	l.Debugf("BonusExp tier for character [%d]: tier=%d, multiplier=%.2f (equippedSince=%v)",
		characterId, tier, multiplier, equippedSince)

	return tier, multiplier
}

// computeWithMidnightRetention handles the case where the item was equipped before midnight
// At midnight, hours reset but the tier is retained until logout or unequip
func computeWithMidnightRetention(l logrus.FieldLogger, ctx context.Context, characterId uint32, equippedSince, todayMidnight, now time.Time, tiers []equipment.BonusExpTier) (int8, float64) {
	// Compute tier at midnight (from sessions between equippedSince and midnight)
	hoursAtMidnight, err := computeEquippedHours(l, ctx, characterId, equippedSince, todayMidnight)
	if err != nil {
		l.WithError(err).Warnf("Failed to compute hours at midnight for character [%d], falling back to 0.", characterId)
		hoursAtMidnight = 0
	}
	tierAtMidnight := getTierFromHours(hoursAtMidnight, tiers)
	multiplierAtMidnight := getMultiplierFromTier(tierAtMidnight, tiers)

	// Compute tier from today's sessions
	hoursSinceMidnight, err := computeEquippedHours(l, ctx, characterId, todayMidnight, now)
	if err != nil {
		l.WithError(err).Warnf("Failed to compute hours since midnight for character [%d], using midnight tier.", characterId)
		return tierAtMidnight, multiplierAtMidnight
	}
	tierToday := getTierFromHours(hoursSinceMidnight, tiers)
	multiplierToday := getMultiplierFromTier(tierToday, tiers)

	// Return the higher tier (midnight retention rule)
	if multiplierAtMidnight > multiplierToday {
		return tierAtMidnight, multiplierAtMidnight
	}
	return tierToday, multiplierToday
}

// computeSimple handles the normal case - equipped after midnight
func computeSimple(l logrus.FieldLogger, ctx context.Context, characterId uint32, equippedSince, now time.Time, tiers []equipment.BonusExpTier) (int8, float64) {
	hours, err := computeEquippedHours(l, ctx, characterId, equippedSince, now)
	if err != nil {
		l.WithError(err).Warnf("Failed to compute equipped hours for character [%d], falling back to 0.", characterId)
		hours = 0
	}

	tier := getTierFromHours(hours, tiers)
	multiplier := getMultiplierFromTier(tier, tiers)
	return tier, multiplier
}

// computeEquippedHours computes total equipped hours within a time range using session history
func computeEquippedHours(l logrus.FieldLogger, ctx context.Context, characterId uint32, start, end time.Time) (int64, error) {
	playtime, err := session.ComputePlaytimeInRange(l)(ctx)(characterId, start, end)
	if err != nil {
		return 0, err
	}
	return int64(playtime.Hours()), nil
}

// getTierFromHours returns the tier index (0-based) for the given hours
func getTierFromHours(hours int64, tiers []equipment.BonusExpTier) int8 {
	var currentTier int8 = 0
	for i, tier := range tiers {
		if hours >= int64(tier.TermStart) {
			currentTier = int8(i + 1)
		}
	}
	return currentTier
}

// getMultiplierFromTier returns the multiplier for the given tier
func getMultiplierFromTier(tier int8, tiers []equipment.BonusExpTier) float64 {
	if tier <= 0 || int(tier-1) >= len(tiers) {
		return 1.0
	}

	// Find the tier with the highest IncExpR up to the current tier
	var maxBonus int32 = 0
	for i := 0; i < int(tier) && i < len(tiers); i++ {
		if tiers[i].IncExpR > maxBonus {
			maxBonus = tiers[i].IncExpR
		}
	}

	if maxBonus <= 0 {
		return 1.0
	}

	// Convert percentage to multiplier (e.g., 30 -> 1.30)
	return 1.0 + (float64(maxBonus) / 100.0)
}

// truncateToDay returns the start of the day (midnight) for the given time
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
