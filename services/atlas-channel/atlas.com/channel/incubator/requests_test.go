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

// TestSelectReward_RoundTrip stands up an httptest server standing in for
// atlas-reward-pools, asserts NewProcessor.SelectReward POSTs to
// gachapons/{eggId}/rewards/select (eggId as a path segment, not a query
// param) and decodes the returned gachapon-rewards resource into a
// populated Reward.
func TestSelectReward_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.True(t, strings.HasSuffix(r.URL.Path, "/gachapons/4170000/rewards/select"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {"type":"gachapon-rewards","id":"2000000","attributes":{"itemId":2000000,"quantity":5,"tier":"common","gachaponId":"4170000"}}
		}`))
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	reward, err := NewProcessor(logrus.New(), ctx).SelectReward(4170000)
	require.NoError(t, err)

	require.Equal(t, uint32(2000000), reward.ItemId())
	require.Equal(t, uint32(5), reward.Quantity())
}

// TestSelectReward_QuantityDefaultsToOne guards the documented quantity
// 0->1 default when atlas-reward-pools omits quantity.
func TestSelectReward_QuantityDefaultsToOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {"type":"gachapon-rewards","id":"2000001","attributes":{"itemId":2000001,"tier":"rare","gachaponId":"4170000"}}
		}`))
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	reward, err := NewProcessor(logrus.New(), ctx).SelectReward(4170000)
	require.NoError(t, err)

	require.Equal(t, uint32(2000001), reward.ItemId())
	require.Equal(t, uint32(1), reward.Quantity(), "quantity 0 must default to 1")
}

// TestSelectReward_InfrastructureError verifies a 5xx from atlas-reward-pools
// surfaces as a non-nil error rather than being silently swallowed into a
// zero-value reward.
func TestSelectReward_InfrastructureError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).SelectReward(4170000)
	require.Error(t, err)
}

// TestSuccessNpcAvailable_Present verifies a 200 from atlas-data's
// /data/npcs/{SuccessNpcId} yields (true, nil) — the client can render the
// result dialog, so incubation may proceed.
func TestSuccessNpcAvailable_Present(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.True(t, strings.HasSuffix(r.URL.Path, "/data/npcs/9050008"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"npcs","id":"9050008","attributes":{"name":"Pigmy"}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	ok, err := NewProcessor(logrus.New(), ctx).SuccessNpcAvailable()
	require.NoError(t, err)
	require.True(t, ok)
}

// TestSuccessNpcAvailable_Missing verifies a 404 yields (false, nil) — the GMS
// case where the fixed incubator NPC (SuccessNpcId) was never shipped, so the
// handler must block instead of letting the client crash.
func TestSuccessNpcAvailable_Missing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	ok, err := NewProcessor(logrus.New(), ctx).SuccessNpcAvailable()
	require.NoError(t, err)
	require.False(t, ok)
}
