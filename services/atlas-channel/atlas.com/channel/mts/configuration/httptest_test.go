package configuration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// mtsConfigResponse is a real JSON:API "mts-configs" single-object response
// mirroring what atlas-tenants marshals for the MTS configuration resource. It
// carries NON-default knob values so a passing assertion proves the api2go
// unmarshal actually ran (the registry's DefaultConfig fallback would yield the
// era defaults instead) — a FakeClient mock would bypass the unmarshal entirely.
const mtsConfigResponse = `{
  "data": { "type": "mts-configs", "id": "55555555-5555-5555-5555-555555555555", "attributes": {
      "listingFee": 9999, "commissionRate": 0.15, "commissionBase": 750,
      "maxActiveListings": 20, "minLevel": 30, "auctionMinHours": 12,
      "auctionMaxHours": 96, "fixedSaleHours": 72, "priceFloor": 250,
      "pageSize": 24, "minBidIncrement": 5 } }
}`

// TestDefaultFetcher_UnmarshalsConfig stands up an httptest server emulating
// atlas-tenants' configuration endpoint and asserts the registry's defaultFetcher
// unmarshals the JSON:API body into a populated Model. TENANTS_SERVICE_URL points
// the RootUrl("TENANTS") client at the server. This is the fetch path
// GetTenantConfig drives on a cache miss.
func TestDefaultFetcher_UnmarshalsConfig(t *testing.T) {
	tenantId := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(mtsConfigResponse))
	}))
	defer srv.Close()

	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	cfg, err := defaultFetcher(logrus.New(), context.Background(), tenantId)
	if err != nil {
		t.Fatalf("defaultFetcher returned error: %v", err)
	}

	wantPath := "/tenants/" + tenantId.String() + "/configurations/mts-configs"
	if gotPath != wantPath {
		t.Errorf("request path: want %q, got %q", wantPath, gotPath)
	}
	if cfg.CommissionRate() != 0.15 {
		t.Errorf("commissionRate = %v, want 0.15 (proves unmarshal, not the 0.07 default)", cfg.CommissionRate())
	}
	if cfg.ListingFee() != 9999 {
		t.Errorf("listingFee = %d, want 9999", cfg.ListingFee())
	}
	if cfg.PriceFloor() != 250 {
		t.Errorf("priceFloor = %d, want 250 (proves unmarshal, not the 110 default)", cfg.PriceFloor())
	}
	if cfg.MaxActiveListings() != 20 {
		t.Errorf("maxActiveListings = %d, want 20", cfg.MaxActiveListings())
	}
	if cfg.PageSize() != 24 {
		t.Errorf("pageSize = %d, want 24", cfg.PageSize())
	}
}
