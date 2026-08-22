package compartment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestCanAccommodate exercises the cross-service round-trip: the client
// POSTs the requested item and quantity to atlas-inventory's accommodation
// endpoint, and decodes the JSON:API response's overall `accommodated` flag.
func TestCanAccommodate(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"inventoryAccommodations","id":"1","attributes":{"accommodated":true,"results":[{"itemId":2000004,"quantity":30,"accommodated":true}]}}}`))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/api/")

	p := NewProcessor(logrus.New(), context.Background())
	ok, err := p.CanAccommodate(1, 2000004, 30)
	require.NoError(t, err)
	require.True(t, ok, "overall accommodated should reflect the response's true")

	require.Equal(t, http.MethodPost, gotMethod)
	require.True(t, strings.HasSuffix(gotPath, "/characters/1/inventory/accommodation"), "path: %s", gotPath)
}

// A false response is decoded as-is; the caller (receiveParcel) is the one
// that turns it into RECV_NO_FREE_SLOTS.
func TestCanAccommodateFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"inventoryAccommodations","id":"1","attributes":{"accommodated":false,"results":[{"itemId":1302000,"quantity":1,"accommodated":false}]}}}`))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/api/")

	p := NewProcessor(logrus.New(), context.Background())
	ok, err := p.CanAccommodate(1, 1302000, 1)
	require.NoError(t, err)
	require.False(t, ok)
}
