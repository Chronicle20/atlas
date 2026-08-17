package tasks

import (
	"atlas-maps/map/character"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// respawnEnvMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since tasks sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type respawnEnvMarkerKey string

// TestProcessMapsWithCharacters_AppliesEnvContextToSpawn pins the review fix:
// this pod's own environment identity must be threaded onto each map key's
// per-tenant context before the monster/reactor spawn goroutines run. A test
// with an identity envContext would still pass if this were dropped --
// decide() would then fail open per FR-1.8 and every live deployment, not
// just this pod's, would spawn.
func TestProcessMapsWithCharacters_AppliesEnvContextToSpawn(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	mk := character.MapKey{Tenant: ten, Field: f}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, respawnEnvMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processMapsWithCharacters(l, context.Background(), []character.MapKey{mk}, func(ctx context.Context, transactionId uuid.UUID, mk character.MapKey) {
		gotMarker = ctx.Value(respawnEnvMarkerKey("marker"))
	}, envContext)

	require.Equal(t, "stamped", gotMarker, "envContext was not applied to the spawn context")
}
