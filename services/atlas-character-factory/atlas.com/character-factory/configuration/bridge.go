package configuration

import (
	"atlas-character-factory/configuration/tenant"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RunBridge republishes the projection snapshot into the package-level
// configuration vars on a ticker, so GetTenantConfig callers see live
// updates. snap returns a fresh copy of the projection State's tenants
// map (pass projection.State.Snapshot). onChange (may be nil) is invoked
// with (prev, next) before each publish so side effects can diff. The
// first publish happens immediately; subsequent publishes fire every
// interval until ctx is canceled.
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
