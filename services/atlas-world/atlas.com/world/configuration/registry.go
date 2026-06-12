package configuration

import (
	"atlas-world/configuration/tenant"
	"atlas-world/rate"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant2 "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var configMu sync.RWMutex
var tenantConfig map[uuid.UUID]tenant.RestModel

// readyCh is closed once PublishSnapshot has populated tenantConfig for
// the first time. Kafka handlers (channel status) may fire before the
// projection catches up; Get* blocks on readyCh instead of the legacy
// log.Fatalf path, bounded by readyTimeout.
var readyCh = make(chan struct{})
var readyOnce sync.Once

const readyTimeout = 60 * time.Second

// ErrNotReady is returned by Get* when the projection has not yet
// published a snapshot within readyTimeout. Transient.
var ErrNotReady = errors.New("configuration: projection snapshot not yet published")

// ErrTenantNotConfigured is returned by GetTenantConfig when the requested
// tenant is absent from a ready snapshot. Persistent.
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

// GetTenantConfigs returns a copy of the full tenant snapshot. Returns
// ErrNotReady before the first PublishSnapshot; otherwise the (possibly
// empty) map. Never log.Fatalf — callers (the boot channel-status sweep)
// log and skip on error.
func GetTenantConfigs() (map[uuid.UUID]tenant.RestModel, error) {
	if err := waitReady(); err != nil {
		return nil, err
	}
	configMu.RLock()
	defer configMu.RUnlock()
	out := make(map[uuid.UUID]tenant.RestModel, len(tenantConfig))
	for k, v := range tenantConfig {
		out[k] = v
	}
	return out, nil
}

// PublishSnapshot replaces the package-level tenant config with the
// snapshot taken from the kafka-backed projection. The first call closes
// readyCh, unblocking any Get* waiters.
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

// SnapshotReady reports whether the first PublishSnapshot has populated the
// tenant config (readyCh closed). Non-blocking — suitable for a /readyz gate
// so readiness reflects actual snapshot availability rather than just Kafka
// catch-up. Once true it stays true (readyCh is closed once, via readyOnce).
func SnapshotReady() bool {
	select {
	case <-readyCh:
		return true
	default:
		return false
	}
}

// initializeRatesFromConfig initializes the rate registry with rates from
// configuration. Called by the bridge onChange hook (configuration.
// ReinitChangedRates) on initial apply and on each tenant config change.
func initializeRatesFromConfig(l logrus.FieldLogger, tenantId uuid.UUID, tc tenant.RestModel) {
	t, err := tenant2.Create(tenantId, tc.Region, tc.MajorVersion, tc.MinorVersion)
	if err != nil {
		l.WithError(err).Errorf("Unable to create tenant model for rate initialization.")
		return
	}

	ctx := tenant2.WithContext(context.Background(), t)
	for worldId, wc := range tc.Worlds {
		rates := rate.NewModel()
		rates = rates.WithRate(rate.TypeExp, wc.GetExpRate())
		rates = rates.WithRate(rate.TypeMeso, wc.GetMesoRate())
		rates = rates.WithRate(rate.TypeItemDrop, wc.GetItemDropRate())
		rates = rates.WithRate(rate.TypeQuestExp, wc.GetQuestExpRate())

		rate.GetRegistry().InitWorldRates(ctx, world.Id(worldId), rates)
		l.Infof("Initialized world [%d] rates from config: exp=%.2f, meso=%.2f, drop=%.2f, quest=%.2f",
			worldId, wc.GetExpRate(), wc.GetMesoRate(), wc.GetItemDropRate(), wc.GetQuestExpRate())
	}
}
