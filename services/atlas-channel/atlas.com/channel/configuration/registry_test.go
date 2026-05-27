package configuration_test

import (
	"testing"
	"time"

	"atlas-channel/configuration"

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
