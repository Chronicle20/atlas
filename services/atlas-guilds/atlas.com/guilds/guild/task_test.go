package guild

import (
	"atlas-guilds/coordinator"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since guild sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type envMarkerKey string

// buildExpiredCoordination initiates a real guild-creation coordination
// through coordinator.Registry (backed by miniredis) and hands back the
// resulting Model, which carries the tenant coordinator.Registry assigned it
// -- coordinator.Model has no exported constructor, so this is the only way
// to build one with valid, tenant-consistent fields from outside the
// coordinator package.
func buildExpiredCoordination(t *testing.T, ten tenant.Model, leaderId uint32) coordinator.Model {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	coordinator.InitRegistry(client)

	ctx := tenant.WithContext(context.Background(), ten)
	ch := channel.NewModel(0, 0)
	require.NoError(t, coordinator.GetRegistry().Initiate(ctx, ch, "TestGuild", leaderId, []uint32{leaderId}))

	time.Sleep(time.Millisecond)
	expired, err := coordinator.GetRegistry().GetExpiredAcrossTenants(0)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	return expired[0]
}

// TestProcessExpiredCoordinationsAppliesEnvContextToAct pins the review fix:
// this pod's own environment identity must be threaded onto the context
// passed to act for each expired guild-creation coordination. Without this,
// decide() would fail open per FR-1.8 and every live deployment, not just
// this pod's, would act on the expired coordination.
func TestProcessExpiredCoordinationsAppliesEnvContextToAct(t *testing.T) {
	l := setupTestLogger(t)
	ten := setupTestTenant(t)
	g := buildExpiredCoordination(t, ten, 100)

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	var gotLeaderId uint32
	processExpiredCoordinations(l, context.Background(), []coordinator.Model{g},
		func(_ logrus.FieldLogger, ctx context.Context, leaderId uint32) error {
			gotMarker = ctx.Value(envMarkerKey("marker"))
			gotLeaderId = leaderId
			return nil
		},
		envContext,
	)

	if gotMarker != "stamped" {
		t.Fatalf("envContext was not applied to the act context: got %v, want \"stamped\"", gotMarker)
	}
	if gotLeaderId != 100 {
		t.Fatalf("want leaderId 100, got %d", gotLeaderId)
	}
}
