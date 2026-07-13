package listing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

// buildPagedListingsResponse renders a minimal JSON:API "listings" page
// (just enough attributes for Extract to succeed) plus the pagination meta
// block requests.DrainProvider reads (meta.page.last drives its iteration
// bound).
func buildPagedListingsResponse(ids []string, total, number, size, last int) string {
	items := make([]string, 0, len(ids))
	for _, id := range ids {
		items = append(items, fmt.Sprintf(
			`{"type":"listings","id":%q,"attributes":{"worldId":1,"itcSn":1,"sellerId":100100,"sellerName":"Bob","saleType":"auction","state":"active","templateId":1302000,"quantity":1}}`,
			id))
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		strings.Join(items, ","), total, number, size, last)
}

// TestBrowseAll_DrainsAllPages stands up an httptest server emulating
// atlas-mts's paginated browse endpoint (300 rows, server page size driven
// by the request's page[size]) and asserts Processor.BrowseAll drains every
// page and returns the complete, non-duplicated 300-row set — the
// replacement for atlas-mts's removed PageSize:-1 "unpaged" escape hatch.
func TestBrowseAll_DrainsAllPages(t *testing.T) {
	const total = 300
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q := r.URL.Query()
		// Assert BrowseAll's filter reached the server and no page params were
		// baked into the base URL by browseUrl (only DrainProvider appends them).
		if got := q.Get("sellerId"); got != "100100" {
			t.Errorf("request missing sellerId=100100 filter, got query %q", r.URL.RawQuery)
		}
		number, _ := strconv.Atoi(q.Get("page[number]"))
		if number == 0 {
			number = 1
		}
		size, _ := strconv.Atoi(q.Get("page[size]"))
		if size == 0 {
			size = 250
		}
		start := (number - 1) * size
		end := start + size
		if end > total {
			end = total
		}
		ids := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			ids = append(ids, fmt.Sprintf("listing-%d", i))
		}
		last := (total + size - 1) / size
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(buildPagedListingsResponse(ids, total, number, size, last)))
	}))
	defer srv.Close()

	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).BrowseAll(1, BrowseFilter{SellerId: 100100})
	if err != nil {
		t.Fatalf("BrowseAll returned error: %v", err)
	}
	if gotPath != "/worlds/1/listings" {
		t.Errorf("request path: want /worlds/1/listings, got %q", gotPath)
	}
	if len(ms) != total {
		t.Fatalf("BrowseAll returned %d listings, want %d (all rows across both pages)", len(ms), total)
	}
	seen := make(map[string]struct{}, total)
	for _, m := range ms {
		seen[m.Id()] = struct{}{}
	}
	if len(seen) != total {
		t.Fatalf("BrowseAll returned %d distinct ids, want %d (duplicate or missing rows across pages)", len(seen), total)
	}
}

// TestBrowse_SinglePageOnly asserts Processor.Browse (the player-facing
// single-page path) returns exactly what one GET returned — it must NOT
// drain — even when the server reports more rows exist via meta.total.
func TestBrowse_SinglePageOnly(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ids := []string{"a", "b", "c", "d", "e"}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(buildPagedListingsResponse(ids, 50, 1, 5, 10)))
	}))
	defer srv.Close()

	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	ms, err := NewProcessor(logrus.New(), context.Background()).Browse(1, BrowseFilter{SellerId: 100100})
	if err != nil {
		t.Fatalf("Browse returned error: %v", err)
	}
	if len(ms) != 5 {
		t.Fatalf("Browse returned %d listings, want exactly 5 (one page, no drain)", len(ms))
	}
	if requestCount != 1 {
		t.Fatalf("Browse issued %d requests, want exactly 1 (single page, no drain)", requestCount)
	}
}
