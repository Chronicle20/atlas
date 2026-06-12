package configuration_test

import (
	"testing"
	"time"

	"atlas-world/configuration"
	"atlas-world/configuration/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Verifies the crash-fix: GetTenantConfig / GetTenantConfigs block until
// PublishSnapshot rather than log.Fatalf-ing the pod, then resolve
// present/absent tenants without crashing.
func TestRegistry_BlocksThenResolvesAndReportsAbsent(t *testing.T) {
	id := uuid.New()
	type result struct {
		cfg tenant.RestModel
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := configuration.GetTenantConfig(id)
		done <- result{c, err}
	}()

	select {
	case r := <-done:
		t.Fatalf("GetTenantConfig returned before PublishSnapshot (cfg=%v, err=%v)", r.cfg, r.err)
	case <-time.After(100 * time.Millisecond):
	}

	configuration.PublishSnapshot(map[uuid.UUID]tenant.RestModel{
		id: {Id: id.String(), Region: "GMS", MajorVersion: 84, MinorVersion: 1},
	})

	select {
	case r := <-done:
		require.NoError(t, r.err)
		require.Equal(t, "GMS", r.cfg.Region)
	case <-time.After(time.Second):
		t.Fatal("GetTenantConfig did not return after PublishSnapshot")
	}

	_, err := configuration.GetTenantConfig(uuid.New())
	require.ErrorIs(t, err, configuration.ErrTenantNotConfigured)

	// GetTenantConfigs returns the populated snapshot, no Fatalf on empty.
	all, err := configuration.GetTenantConfigs()
	require.NoError(t, err)
	require.Contains(t, all, id)
}
