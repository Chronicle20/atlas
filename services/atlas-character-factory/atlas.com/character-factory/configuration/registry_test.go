package configuration_test

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Verifies the crash-fix: GetTenantConfig blocks until PublishSnapshot
// runs (rather than log.Fatalf-ing the pod), then resolves present and
// absent tenants without crashing.
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

	// Before any PublishSnapshot, GetTenantConfig must block.
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

	// Absent tenant in a ready snapshot → ErrTenantNotConfigured, no crash.
	_, err := configuration.GetTenantConfig(uuid.New())
	require.ErrorIs(t, err, configuration.ErrTenantNotConfigured)
}
