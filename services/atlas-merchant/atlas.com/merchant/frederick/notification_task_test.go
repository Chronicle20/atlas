package frederick

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// frederickEnvMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since frederick sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type frederickEnvMarkerKey string

// TestProcessDueNotifications_AppliesEnvContextToNotify pins the review fix:
// this pod's own environment identity must be threaded onto each due
// notification's per-tenant context before the emit. A test with an
// identity envContext would still pass if this were dropped -- decide()
// would then fail open per FR-1.8 and every live deployment, not just this
// pod's, would notify.
func TestProcessDueNotifications_AppliesEnvContextToNotify(t *testing.T) {
	l, _ := test.NewNullLogger()
	n := NotificationEntity{
		Id:           uuid.New(),
		TenantId:     uuid.New(),
		TenantRegion: "GMS",
		TenantMajor:  83,
		TenantMinor:  1,
		CharacterId:  1000,
		NextDay:      2,
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, frederickEnvMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processDueNotifications(l, context.Background(), []NotificationEntity{n}, func(ctx context.Context, _ NotificationEntity) {
		gotMarker = ctx.Value(frederickEnvMarkerKey("marker"))
	}, envContext)

	require.Equal(t, "stamped", gotMarker, "envContext was not applied to the notify context")
}
