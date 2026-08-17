package expiration

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// IsExpired checks if an item has expired
// Returns true if:
// - Expiration time is set (not zero value)
// - Current time is after expiration time
func IsExpired(expiration time.Time, now time.Time) bool {
	// Zero time means no expiration. Use Equal to handle timezone-shifted
	// zero times (e.g., api2go may serialize zero time with a non-UTC location,
	// causing IsZero() to return false on round-trip).
	if expiration.IsZero() || expiration.Equal(time.Time{}) {
		return false
	}
	return now.After(expiration)
}

// HasExpiration checks if an item has an expiration time set
func HasExpiration(expiration time.Time) bool {
	return !expiration.IsZero() && !expiration.Equal(time.Time{})
}

// IsReapable reports whether an expired asset may be destroyed. Pets are the
// sole exemption: an expired pet does not vanish, it dries up into a doll that
// keeps its cash-inventory slot until a Water of Life (5180000) revives it.
// The rule is by classification, not an id allowlist, so every present and
// future 5000xxx pet template is covered without further edits.
func IsReapable(templateId uint32) bool {
	return item.GetClassification(item.Id(templateId)) != item.ClassificationPet
}
