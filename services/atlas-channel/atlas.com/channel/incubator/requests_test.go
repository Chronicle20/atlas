package incubator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tm
}

// TestGetRewards_RoundTrip stands up an httptest server returning a canned
// incubator-rewards JSON:API document and asserts NewProcessor.GetRewards
// decodes it via requestRewards + Extract into a populated []Reward. It also
// guards the documented quantity 0->1 default: the second reward entry omits
// quantity and must come back as 1, not 0.
func TestGetRewards_RoundTrip(t *testing.T) {
	tm := newTestTenant(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/tenants/"+tm.Id().String()+"/configurations/incubator-rewards"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"type":"incubator-rewards","id":"1","attributes":{"itemId":2000000,"quantity":5,"weight":10,"eggId":4170000}},
				{"type":"incubator-rewards","id":"2","attributes":{"itemId":2000001,"quantity":0,"weight":20,"eggId":4170001}}
			]
		}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/api/")

	ctx := tenant.WithContext(context.Background(), tm)
	rewards, err := NewProcessor(logrus.New(), ctx).GetRewards()
	require.NoError(t, err)
	require.Len(t, rewards, 2)

	require.Equal(t, uint32(2000000), rewards[0].ItemId())
	require.Equal(t, uint32(5), rewards[0].Quantity())
	require.Equal(t, uint32(10), rewards[0].Weight())
	require.Equal(t, uint32(4170000), rewards[0].EggId(), "eggId must round-trip through requestRewards + Extract")

	require.Equal(t, uint32(2000001), rewards[1].ItemId())
	require.Equal(t, uint32(1), rewards[1].Quantity(), "quantity 0 must default to 1")
	require.Equal(t, uint32(20), rewards[1].Weight())
	require.Equal(t, uint32(4170001), rewards[1].EggId(), "eggId must round-trip through requestRewards + Extract")
}

// TestGetRewards_InfrastructureError verifies a 5xx from atlas-tenants
// surfaces as a non-nil error rather than being silently swallowed into an
// empty reward pool.
func TestGetRewards_InfrastructureError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).GetRewards()
	require.Error(t, err)
}
