package merchant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestExtractSearchListing(t *testing.T) {
	shopId := uuid.New()
	rm := ListingSearchRestModel{
		Id:               uuid.New().String(),
		ShopId:           shopId.String(),
		ShopTitle:        "cheap stuff",
		WorldId:          0,
		ChannelId:        2,
		MapId:            910000004,
		OwnerId:          30001,
		ShopType:         1,
		State:            StateOpen,
		ItemId:           2060000,
		ItemType:         2,
		Quantity:         100,
		BundleSize:       100,
		BundlesRemaining: 3,
		PricePerBundle:   5000,
		ItemSnapshot:     SnapshotRestModel{Quantity: 100},
	}
	m, err := ExtractSearchListing(rm)
	require.NoError(t, err)
	require.Equal(t, shopId, m.ShopId())
	require.Equal(t, "cheap stuff", m.Title())
	require.Equal(t, uint32(30001), m.OwnerId())
	require.Equal(t, byte(1), m.ShopType())
	require.Equal(t, StateOpen, m.State())
	require.Equal(t, uint32(910000004), m.MapId())
	require.Equal(t, uint16(3), m.BundlesRemaining())
	require.Equal(t, uint32(5000), m.PricePerBundle())
	require.Equal(t, uint32(100), m.ItemSnapshot().Quantity)
}

func TestExtractTopSearch(t *testing.T) {
	m, err := ExtractTopSearch(TopSearchRestModel{Id: "2060000", ItemId: 2060000, Count: 42})
	require.NoError(t, err)
	require.Equal(t, uint32(2060000), m.ItemId())
	require.Equal(t, uint64(42), m.Count())
}

// TestHasFrederickPending exercises the real cross-service path — URL
// construction, JSON:API unmarshal of atlas-merchant's frederick-status
// resource — against an httptest server (the mock-based tests bypass both).
func TestHasFrederickPending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/characters/1000/frederick" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"frederick-status","id":"1000","attributes":{"hasPending":true}}}`))
	}))
	defer srv.Close()
	t.Setenv("MERCHANT_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	got, err := NewProcessor(l, ctx).HasFrederickPending(1000)
	require.NoError(t, err)
	require.True(t, got)
}
