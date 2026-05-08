package lock

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestOptions_DefaultsApplied(t *testing.T) {
	cfg := config{}
	applyDefaults(&cfg)
	require.Equal(t, 30*time.Second, cfg.ttl)
	require.Equal(t, 10*time.Second, cfg.refreshInterval)
	require.Equal(t, 5*time.Second, cfg.backoff)
	require.Equal(t, 5*time.Second, cfg.gracePeriod)
	require.NotNil(t, cfg.log)
}

func TestOptions_OverridesApplied(t *testing.T) {
	cfg := config{}
	applyDefaults(&cfg)
	WithTTL(2 * time.Minute)(&cfg)
	WithRefreshInterval(20 * time.Second)(&cfg)
	WithBackoff(15 * time.Second)(&cfg)
	WithGracePeriod(10 * time.Second)(&cfg)
	l := logrus.New()
	WithLogger(l)(&cfg)

	require.Equal(t, 2*time.Minute, cfg.ttl)
	require.Equal(t, 20*time.Second, cfg.refreshInterval)
	require.Equal(t, 15*time.Second, cfg.backoff)
	require.Equal(t, 10*time.Second, cfg.gracePeriod)
	require.Same(t, l, cfg.log)
}
