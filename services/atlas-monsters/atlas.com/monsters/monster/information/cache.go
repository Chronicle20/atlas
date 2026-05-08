package information

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// --- Configuration ---------------------------------------------------------

type cacheConfig struct {
	enabled     bool
	ttl         time.Duration
	negativeTTL time.Duration
}

const (
	envEnabled     = "MONSTER_DATA_CACHE_ENABLED"
	envTTL         = "MONSTER_DATA_CACHE_TTL"
	envNegativeTTL = "MONSTER_DATA_CACHE_NEGATIVE_TTL"

	defaultTTL         = 5 * time.Minute
	defaultNegativeTTL = 30 * time.Second

	minTTL         = 1 * time.Second
	maxTTL         = 24 * time.Hour
	minNegativeTTL = 0 * time.Second
	maxNegativeTTL = 5 * time.Minute
)

// configLogger is the logger used for one-time configuration warnings.
// Tests may swap it; in production it stays the standard logger.
var configLogger logrus.FieldLogger = logrus.StandardLogger()

func loadConfig() cacheConfig {
	return cacheConfig{
		enabled:     parseBoolEnv(envEnabled, true),
		ttl:         parseDurationEnv(envTTL, defaultTTL, minTTL, maxTTL),
		negativeTTL: parseDurationEnv(envNegativeTTL, defaultNegativeTTL, minNegativeTTL, maxNegativeTTL),
	}
}

func parseBoolEnv(name string, def bool) bool {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	switch v {
	case "true", "TRUE", "True", "1", "yes", "y":
		return true
	case "false", "FALSE", "False", "0", "no", "n":
		return false
	default:
		configLogger.Warnf("invalid bool for %s=%q; using default %v", name, v, def)
		return def
	}
}

func parseDurationEnv(name string, def, lo, hi time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		configLogger.Warnf("invalid duration for %s=%q; using default %s", name, v, def)
		return def
	}
	if d < lo || d > hi {
		configLogger.Warnf("%s=%s out of range [%s, %s]; using default %s", name, d, lo, hi, def)
		return def
	}
	return d
}

// --- Error classification --------------------------------------------------

type errKind int

const (
	errKindTransient errKind = iota
	errKindNotFound
)

// classifyError decides whether a failed upstream lookup should be cached
// as a negative entry. The transport at libs/atlas-rest/requests returns
// the sentinel requests.ErrNotFound on HTTP 404 (libs/atlas-rest/requests/get.go:14-15);
// everything else (network, 5xx, parse, retry exhaustion, ErrBadRequest) is
// treated as transient and not cached.
func classifyError(err error) errKind {
	if errors.Is(err, requests.ErrNotFound) {
		return errKindNotFound
	}
	return errKindTransient
}

// notFoundError synthesizes a not-found error for negative-cache hits so
// callers see the same errors.Is(err, requests.ErrNotFound) shape they
// would see from a live miss.
func notFoundError(monsterId uint32) error {
	return fmt.Errorf("monster %d not found: %w", monsterId, requests.ErrNotFound)
}
