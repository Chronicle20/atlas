package configuration_test

import (
	"atlas-channel/configuration"
	"atlas-channel/configuration/tenant"
	"atlas-channel/configuration/tenant/diagnostics"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Verifies the latent startup race fix: Get* blocks until PublishSnapshot
// runs, rather than crashing the pod via log.Fatal. Mirrors the failure
// mode that crashed atlas-login on PR 522.
func TestGetServiceConfig_BlocksUntilPublishSnapshot(t *testing.T) {
	type result struct {
		cfg *configuration.RestModel
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := configuration.GetServiceConfig()
		done <- result{c, err}
	}()

	select {
	case r := <-done:
		t.Fatalf("GetServiceConfig returned before PublishSnapshot (cfg=%v, err=%v)", r.cfg, r.err)
	case <-time.After(100 * time.Millisecond):
	}

	id := uuid.New()
	configuration.PublishSnapshot(&configuration.RestModel{Id: id}, nil)

	select {
	case r := <-done:
		require.NoError(t, r.err)
		require.NotNil(t, r.cfg)
		require.Equal(t, id, r.cfg.Id)
	case <-time.After(time.Second):
		t.Fatal("GetServiceConfig did not return after PublishSnapshot")
	}
}

// TestTracePacketsEnabled verifies TracePacketsEnabled never blocks (FR-2.4)
// and correctly reflects a per-tenant flag, including a live change
// (FR-2.3) without leaking to a sibling tenant (FR-2.6). Subtests run in
// declared order (none is t.Parallel), and each publishes its own snapshot
// with fresh tenant ids so it does not depend on a prior subtest's
// published state -- only on PublishSnapshot's readyCh having been closed
// at least once, which is idempotent across the package.
//
// The first subtest never asserts that TracePacketsEnabled blocks, only
// that it returns promptly, because this file also contains
// TestGetServiceConfig_BlocksUntilPublishSnapshot, which also publishes, so
// tests may run in either order across the package.
func TestTracePacketsEnabled(t *testing.T) {
	t.Run("never blocks, even before this test's own snapshot", func(t *testing.T) {
		done := make(chan bool, 1)
		go func() {
			done <- configuration.TracePacketsEnabled(uuid.New())
		}()
		select {
		case v := <-done:
			require.False(t, v)
		case <-time.After(50 * time.Millisecond):
			t.Fatal("TracePacketsEnabled blocked")
		}
	})

	t.Run("tenant absent from the snapshot", func(t *testing.T) {
		tenantA := uuid.New()
		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tenantA: {},
		})
		require.False(t, configuration.TracePacketsEnabled(uuid.New()))
	})

	t.Run("flag on for A, off for B", func(t *testing.T) {
		tenantA := uuid.New()
		tenantB := uuid.New()
		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: true}},
			tenantB: {},
		})
		require.True(t, configuration.TracePacketsEnabled(tenantA))
		require.False(t, configuration.TracePacketsEnabled(tenantB))
	})

	t.Run("live change -- flipping A back off takes effect immediately", func(t *testing.T) {
		tenantA := uuid.New()
		tenantB := uuid.New()
		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: true}},
			tenantB: {},
		})
		require.True(t, configuration.TracePacketsEnabled(tenantA))

		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: false}},
			tenantB: {},
		})
		require.False(t, configuration.TracePacketsEnabled(tenantA))
	})
}
