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
// (FR-2.3) without leaking to a sibling tenant (FR-2.6). Written as one
// function with sequential phases, not parallel subtests, because
// PublishSnapshot mutates shared package state.
//
// Phase 1 must run first (relative to this test's own logic) because
// PublishSnapshot closes readyCh irreversibly. This file also contains
// TestGetServiceConfig_BlocksUntilPublishSnapshot, which also publishes, so
// tests may run in either order across the package -- phase 1 therefore
// never asserts that TracePacketsEnabled blocks, only that it returns
// promptly, regardless of whether a snapshot has already been published by
// another test.
func TestTracePacketsEnabled(t *testing.T) {
	// Phase 1: before any snapshot from this test, a lookup must return
	// promptly (never block on readyCh) and must not panic on a nil map.
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

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Phase 2: tenant absent from the snapshot.
	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tenantA: {},
	})
	require.False(t, configuration.TracePacketsEnabled(uuid.New()))

	// Phase 3: flag on for A, off for B.
	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: true}},
		tenantB: {},
	})
	require.True(t, configuration.TracePacketsEnabled(tenantA))
	require.False(t, configuration.TracePacketsEnabled(tenantB))

	// Phase 4: live change -- flipping A back off takes effect immediately.
	configuration.PublishSnapshot(&configuration.RestModel{Id: uuid.New()}, map[uuid.UUID]tenant.RestModel{
		tenantA: {Diagnostics: diagnostics.RestModel{TracePackets: false}},
		tenantB: {},
	})
	require.False(t, configuration.TracePacketsEnabled(tenantA))
}
