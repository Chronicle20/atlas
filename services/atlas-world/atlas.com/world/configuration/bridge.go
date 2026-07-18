package configuration

import (
	"atlas-world/configuration/tenant"
	"context"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RunBridge republishes the projection snapshot into the package-level
// configuration vars on a ticker, so GetTenantConfig(s) callers see live
// updates. snap returns a fresh copy of the projection State's tenants
// map (pass projection.State.Snapshot). onChange (may be nil) is invoked
// with (prev, next) before each publish so side effects (rate re-init)
// can diff. The first publish happens immediately; subsequent publishes
// fire every interval until ctx is canceled.
func RunBridge(
	ctx context.Context,
	l logrus.FieldLogger,
	snap func() map[uuid.UUID]tenant.RestModel,
	interval time.Duration,
	onChange func(prev, next map[uuid.UUID]tenant.RestModel),
) {
	var prev map[uuid.UUID]tenant.RestModel
	publish := func() {
		next := snap()
		if onChange != nil {
			onChange(prev, next)
		}
		PublishSnapshot(next)
		prev = next
	}
	publish()

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			publish()
		}
	}
}

// ReinitChangedRates returns an onChange hook that re-initializes world
// rates for tenants whose config newly appeared or changed since the
// previous snapshot. Unchanged tenants are left untouched so live
// SetWorldRate overrides survive between config changes (design Q1). A
// config change clobbers that tenant's live rates from config.
func ReinitChangedRates(l logrus.FieldLogger) func(prev, next map[uuid.UUID]tenant.RestModel) {
	return func(prev, next map[uuid.UUID]tenant.RestModel) {
		for _, id := range changedTenants(prev, next) {
			initializeRatesFromConfig(l, id, next[id])
		}
	}
}

// changedTenants returns the ids in next that are absent from prev or
// whose config differs by value. Removed tenants (in prev, absent from
// next) are not returned and never cause a panic.
func changedTenants(prev, next map[uuid.UUID]tenant.RestModel) []uuid.UUID {
	var out []uuid.UUID
	for id, nc := range next {
		pc, ok := prev[id]
		if !ok || !reflect.DeepEqual(pc, nc) {
			out = append(out, id)
		}
	}
	return out
}
