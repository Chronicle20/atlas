package listing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// listingsResponse is a real JSON:API "listings" list response mirroring what
// atlas-mts's browse handler marshals. Serving it verbatim drives the same api2go
// unmarshal path (and the SetToOneReferenceID/SetToManyReferenceIDs relationship
// stubs) the live client uses — a FakeClient mock would bypass it.
const listingsResponse = `{
  "data": [
    { "type": "listings", "id": "22222222-2222-2222-2222-222222222222", "attributes": {
        "worldId": 1, "itcSn": 4242, "sellerId": 100100, "sellerName": "Bob",
        "saleType": "auction", "state": "active",
        "templateId": 1302000, "quantity": 1,
        "listValue": 5000, "buyNowPrice": 9000, "category": "equip", "subCategory": "onehand",
        "currentBid": 5500, "highBidderId": 200200, "minIncrement": 100, "bidCount": 2 } }
  ]
}`

// TestBrowse_UnmarshalsListing stands up an httptest server emulating atlas-mts's
// browse endpoint and asserts Processor.Browse unmarshals the JSON:API body into a
// populated Model. MTS_SERVICE_URL points the RootUrl("MTS") client at the server.
func TestBrowse_UnmarshalsListing(t *testing.T) {
	var gotPath, gotSeller string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSeller = r.URL.Query().Get("sellerName")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(listingsResponse))
	}))
	defer srv.Close()

	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).Browse(1, BrowseFilter{SellerName: "Bob"})
	if err != nil {
		t.Fatalf("Browse returned error: %v", err)
	}

	if gotPath != "/worlds/1/listings" {
		t.Errorf("request path: want /worlds/1/listings, got %q", gotPath)
	}
	if gotSeller != "Bob" {
		t.Errorf("sellerName query: want Bob, got %q", gotSeller)
	}
	if len(ms) != 1 {
		t.Fatalf("listings: want 1, got %d", len(ms))
	}
	m := ms[0]
	if m.Id() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("id = %q", m.Id())
	}
	if m.ListValue() != 5000 {
		t.Errorf("listValue = %d, want 5000", m.ListValue())
	}
	if m.TemplateId() != 1302000 {
		t.Errorf("templateId = %d, want 1302000", m.TemplateId())
	}
	if m.CurrentBid() != 5500 {
		t.Errorf("currentBid = %d, want 5500", m.CurrentBid())
	}
	if m.SellerName() != "Bob" {
		t.Errorf("sellerName = %q, want Bob", m.SellerName())
	}
}
