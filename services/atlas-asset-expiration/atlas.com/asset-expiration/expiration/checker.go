package expiration

import (
	"time"
)

// IsExpired checks if an item has expired
// Returns true if:
// - Expiration time is set (not zero value)
// - Current time is after expiration time
func IsExpired(expiration time.Time, now time.Time) bool {
	// Zero time means no expiration
	if expiration.IsZero() {
		return false
	}
	return now.After(expiration)
}

// HasExpiration checks if an item has an expiration time set
func HasExpiration(expiration time.Time) bool {
	return !expiration.IsZero()
}
