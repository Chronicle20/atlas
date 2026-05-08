package cache

import "time"

// Config configures a Cache instance.
type Config struct {
	// TTL is the lifetime of a positive entry. Must be > 0.
	TTL time.Duration

	// NegativeTTL is the lifetime of a negative entry. Zero disables
	// negative caching (PutNegative is a no-op; IsNegative always
	// returns false).
	NegativeTTL time.Duration

	// Now is the clock function. nil falls back to time.Now.
	Now func() time.Time

	// OnEviction is called under the cache's write lock when a lazy
	// expiration removes an entry. nil disables the callback. kind is
	// "positive" or "negative".
	OnEviction func(kind string)
}
