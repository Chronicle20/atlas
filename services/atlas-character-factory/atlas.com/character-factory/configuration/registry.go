package configuration

import (
	"atlas-character-factory/configuration/tenant"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	configMu     sync.RWMutex
	tenantConfig map[uuid.UUID]tenant.RestModel
)

// readyCh is closed once PublishSnapshot has populated tenantConfig for
// the first time. Kafka handlers (the seed saga) may fire before the
// projection catches up; GetTenantConfig blocks on readyCh instead of the
// legacy log.Fatalf path, bounded by readyTimeout.
var (
	readyCh   = make(chan struct{})
	readyOnce sync.Once
)

// readyTimeout caps how long GetTenantConfig waits for the projection's
// first PublishSnapshot. Long enough to outlast the catch-up window in a
// fresh PR env, short enough that a wedged projection surfaces as request
// errors rather than goroutine pileup.
const readyTimeout = 60 * time.Second

// ErrNotReady is returned by GetTenantConfig when the projection has not
// yet published a snapshot within readyTimeout. Transient: callers should
// log at DEBUG and skip; /readyz keeps the pod out of service until
// catch-up completes.
var ErrNotReady = errors.New("configuration: projection snapshot not yet published")

// ErrTenantNotConfigured is returned by GetTenantConfig when the requested
// tenant is absent from a ready snapshot. Persistent (vs ErrNotReady) — a
// tenant that was never in the projection won't appear by waiting.
var ErrTenantNotConfigured = errors.New("configuration: tenant not configured")

func waitReady() error {
	select {
	case <-readyCh:
		return nil
	case <-time.After(readyTimeout):
		return ErrNotReady
	}
}

func GetTenantConfig(tenantId uuid.UUID) (tenant.RestModel, error) {
	if err := waitReady(); err != nil {
		return tenant.RestModel{}, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	val, ok := tenantConfig[tenantId]
	if !ok {
		return tenant.RestModel{}, ErrTenantNotConfigured
	}
	return val, nil
}

// PublishSnapshot replaces the package-level tenant config with the
// snapshot taken from the kafka-backed projection. Called by the bridge
// (configuration.RunBridge) after CaughtUp fires and on each observed
// change so legacy GetTenantConfig callers see the same data. The map is
// copied by value so the caller's projection State can mutate
// independently after the call. The first call closes readyCh,
// unblocking any GetTenantConfig waiters.
func PublishSnapshot(tenants map[uuid.UUID]tenant.RestModel) {
	configMu.Lock()
	next := make(map[uuid.UUID]tenant.RestModel, len(tenants))
	for k, v := range tenants {
		next[k] = v
	}
	tenantConfig = next
	configMu.Unlock()

	readyOnce.Do(func() { close(readyCh) })
}
