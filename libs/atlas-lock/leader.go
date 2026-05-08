package lock

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Defaults exposed for documentation; consumers should pass options explicitly.
const (
	DefaultTTL             = 30 * time.Second
	DefaultRefreshInterval = 10 * time.Second // TTL / 3
	DefaultBackoff         = 5 * time.Second
	DefaultGracePeriod     = 5 * time.Second
)

type config struct {
	ttl             time.Duration
	refreshInterval time.Duration
	backoff         time.Duration
	gracePeriod     time.Duration
	log             logrus.FieldLogger
}

// Option mutates a config. Use the WithXxx constructors to obtain Options.
type Option func(*config)

// WithTTL sets the lease TTL. Allowed range: [5s, 5m]. Default: 30s.
func WithTTL(d time.Duration) Option { return func(c *config) { c.ttl = d } }

// WithRefreshInterval sets the renewal cadence. Allowed range: [1s, TTL/2]. Default: TTL/3.
func WithRefreshInterval(d time.Duration) Option {
	return func(c *config) { c.refreshInterval = d }
}

// WithBackoff sets the wait between failed acquire attempts. Allowed range: [1s, 1m]. Default: 5s.
func WithBackoff(d time.Duration) Option { return func(c *config) { c.backoff = d } }

// WithGracePeriod sets how long Run waits for fn to return after lease loss
// before logging a warning and proceeding. Allowed range: [1s, 30s]. Default: 5s.
func WithGracePeriod(d time.Duration) Option { return func(c *config) { c.gracePeriod = d } }

// WithLogger overrides the default logrus.New() logger.
func WithLogger(l logrus.FieldLogger) Option { return func(c *config) { c.log = l } }

func applyDefaults(c *config) {
	c.ttl = DefaultTTL
	c.refreshInterval = DefaultRefreshInterval
	c.backoff = DefaultBackoff
	c.gracePeriod = DefaultGracePeriod
	c.log = logrus.New()
}
