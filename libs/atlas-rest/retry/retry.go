package retry

import (
	"context"

	retry2 "github.com/Chronicle20/atlas-retry"
)

// Config re-exports the shared retry configuration.
type Config = retry2.Config

// DefaultConfig re-exports the shared default configuration.
var DefaultConfig = retry2.DefaultConfig

// Try executes fn with exponential backoff and full jitter.
var Try = retry2.Try

// LegacyTry preserves the old (fn, retries) signature with fixed 1-second sleep
// and no backoff, for callers that have not yet been updated.
func LegacyTry(fn func(attempt int) (bool, error), retries int) error {
	cfg := retry2.Config{
		MaxRetries:    retries,
		InitialDelay:  1e9, // 1 second
		MaxDelay:      1e9,
		BackoffFactor: 1.0,
	}
	return retry2.Try(context.Background(), cfg, fn)
}
