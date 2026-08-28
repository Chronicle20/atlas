package configuration_test

import (
	"atlas-login/configuration"
	"atlas-login/configuration/tenant"
	"atlas-login/configuration/tenant/diagnostics"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Verifies the startup race fix: Get* blocks until PublishSnapshot runs,
// rather than crashing the pod via log.Fatal. Reproduces the failure mode
// observed in PR 522 where atlas-login restarted 3× because Kafka consumer
// handlers fired before configuration.PublishSnapshot populated the
// package-level vars.
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
// (FR-2.3) without leaking to a sibling tenant (FR-2.6). Each subtest
// publishes a snapshot scoped to tenant ids it minted itself (uuid.New()),
// so PublishSnapshot's full-map replace in one subtest can never leak a
// flag into another subtest's lookup.
func TestTracePacketsEnabled(t *testing.T) {
	t.Run("returns promptly without blocking on an unpublished tenant (FR-2.4)", func(t *testing.T) {
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

	t.Run("flag on for one tenant, off for sibling (FR-2.6)", func(t *testing.T) {
		tenantA := uuid.New()
		tenantB := uuid.New()
		configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
			tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: true}},
			tenantB: {},
		})
		require.True(t, configuration.TracePacketsEnabled(tenantA))
		require.False(t, configuration.TracePacketsEnabled(tenantB))
	})

	t.Run("live change -- flipping the flag back off takes effect immediately (FR-2.3)", func(t *testing.T) {
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
