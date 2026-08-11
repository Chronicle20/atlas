package rewardpool

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tm
}

// TestSelectRewardReturnsCommodity verifies a 200 from atlas-reward-pools
// with a gachapon-rewards resource decodes into a Model carrying itemId,
// quantity, and commodityId.
func TestSelectRewardReturnsCommodity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.True(t, strings.HasSuffix(r.URL.Path, "/gachapons/5222000/rewards/select"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {"type":"gachapon-rewards","id":"5222001","attributes":{"itemId":5222001,"quantity":1,"commodityId":40000,"gachaponId":"5222000"}}
		}`))
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	m, err := NewProcessor(logrus.New(), ctx).SelectReward(5222000)
	require.NoError(t, err)

	require.Equal(t, uint32(5222001), m.ItemId())
	require.Equal(t, uint32(1), m.Quantity())
	require.Equal(t, uint32(40000), m.CommodityId())
}

// TestSelectRewardMissingPoolMapsToErrPoolMissing verifies a 404 (no
// cash-surprise pool configured for this box template id) maps to
// ErrPoolMissing.
func TestSelectRewardMissingPoolMapsToErrPoolMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).SelectReward(5222000)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrPoolMissing), "expected ErrPoolMissing, got %v", err)
	require.False(t, errors.Is(err, ErrPoolEmpty), "must not also be ErrPoolEmpty")
}

// TestSelectRewardEmptyPoolMapsToErrPoolEmpty verifies a 409 (pool exists
// but has no eligible entries, task-207 FR-3.7) maps to ErrPoolEmpty.
func TestSelectRewardEmptyPoolMapsToErrPoolEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).SelectReward(5222000)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrPoolEmpty), "expected ErrPoolEmpty, got %v", err)
	require.False(t, errors.Is(err, ErrPoolMissing), "must not also be ErrPoolMissing")
}

// TestSelectRewardTransportFailureIsNotSwallowed verifies an infrastructure
// fault (500) is reported as a plain error that is NEITHER ErrPoolMissing
// NOR ErrPoolEmpty — an operator must not go looking for a misconfigured
// pool when the real problem is atlas-reward-pools itself being down.
func TestSelectRewardTransportFailureIsNotSwallowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("GACHAPONS_SERVICE_URL", srv.URL+"/api/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).SelectReward(5222000)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrPoolMissing), "500 must not be reported as ErrPoolMissing")
	require.False(t, errors.Is(err, ErrPoolEmpty), "500 must not be reported as ErrPoolEmpty")
}
