package tasks

import (
	"atlas-maps/map/weather"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// weatherEnvMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since tasks sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type weatherEnvMarkerKey string

// TestProcessExpiredWeather_AppliesEnvContextToEmit pins the review fix:
// this pod's own environment identity must be threaded onto each expired
// weather entry's per-tenant context before the weather-end emit. A test
// with an identity envContext would still pass if this were dropped --
// decide() would then fail open per FR-1.8 and every live deployment, not
// just this pod's, would react to the weather end.
func TestProcessExpiredWeather_AppliesEnvContextToEmit(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	key := weather.FieldKey{Tenant: ten, Field: f}
	entry := weather.ExpiredEntry{
		Key:   key,
		Entry: weather.WeatherEntry{ItemId: 5, Message: "test", ExpiresAt: time.Now().Add(-time.Second)},
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, weatherEnvMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processExpiredWeather(l, context.Background(), []weather.ExpiredEntry{entry}, func(ctx context.Context, e weather.ExpiredEntry) error {
		gotMarker = ctx.Value(weatherEnvMarkerKey("marker"))
		return nil
	}, envContext)

	require.Equal(t, "stamped", gotMarker, "envContext was not applied to the weather-end emit context")
}
