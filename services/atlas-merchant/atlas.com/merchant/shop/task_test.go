package shop

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// shopEnvMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since shop sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type shopEnvMarkerKey string

// TestProcessExpiredShops_AppliesEnvContextToClose pins the review fix: this
// pod's own environment identity must be threaded onto each expired shop's
// per-tenant context before the close-and-emit. A test with an identity
// envContext would still pass if this were dropped -- decide() would then
// fail open per FR-1.8 and every live deployment, not just this pod's,
// would close the shop.
func TestProcessExpiredShops_AppliesEnvContextToClose(t *testing.T) {
	l, _ := test.NewNullLogger()
	e := Entity{
		Id:           uuid.New(),
		TenantId:     uuid.New(),
		TenantRegion: "GMS",
		TenantMajor:  83,
		TenantMinor:  1,
		CharacterId:  1000,
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, shopEnvMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processExpiredShops(l, context.Background(), []Entity{e}, func(ctx context.Context, _ Entity) error {
		gotMarker = ctx.Value(shopEnvMarkerKey("marker"))
		return nil
	}, envContext)

	require.Equal(t, "stamped", gotMarker, "envContext was not applied to the close-shop context")
}
