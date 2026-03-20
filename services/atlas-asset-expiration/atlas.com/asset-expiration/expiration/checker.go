package expiration

import (
	"time"
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
