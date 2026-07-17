package listing_test

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// testServerInfo implements jsonapi.ServerInformation for the resource tests.
type testServerInfo struct{}

func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t testServerInfo) GetPrefix() string  { return "/api" }

// newListingServer wires the listing routes onto an httptest server backed by
// the shared in-memory test DB.
func newListingServer(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	listing.InitResource(testServerInfo{})(db)(router, l)
	return httptest.NewServer(router)
}

// withTenant builds a request carrying the fixed test tenant headers so the
// GORM tenant callback scopes to the same tenant the processor seeded rows under.
func withTenant(t *testing.T, method, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("TENANT_ID", test.TestTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedListing(t *testing.T, p listing.Processor, worldId world.Id, sellerId uint32, category string, saleType listing.SaleType) listing.Model {
	t.Helper()
	buyNow := uint32(5000)
	m, err := listing.NewBuilder(test.TestTenantId, worldId, sellerId).
		SetSellerName("Seller").
		SetSaleType(saleType).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetBuyNowPrice(&buyNow).
		SetCommissionRate(0.05).
		SetCategory(category).
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("build listing: %v", err)
	}
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}
	return created
}

// TestBrowseEnvelopeAndFilter asserts the browse endpoint returns a JSON:API
// envelope (data array of "listings"), excludes non-active listings, and that a
// category filter actually narrows the result set.
func TestBrowseEnvelopeAndFilter(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	seedListing(t, p, 0, 100, "equip", listing.SaleTypeFixed)
	seedListing(t, p, 0, 101, "use", listing.SaleTypeFixed)
	seedListing(t, p, 1, 102, "equip", listing.SaleTypeFixed)

	srv := newListingServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	// Browse world 0 (no filter) => 2 active listings.
	resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings", srv.URL)))
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("browse status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data []struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				WorldId  byte   `json:"worldId"`
				Category string `json:"category"`
				State    string `json:"state"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode browse: %v", err)
	}
	resp.Body.Close()
	if len(env.Data) != 2 {
		t.Fatalf("browse(w0) returned %d listings, want 2", len(env.Data))
	}
	for _, d := range env.Data {
		if d.Type != "listings" {
			t.Errorf("resource type = %q, want listings", d.Type)
		}
		if d.Attributes.State != "active" {
			t.Errorf("browse returned non-active state %q", d.Attributes.State)
		}
		if d.Attributes.WorldId != 0 {
			t.Errorf("browse(w0) returned worldId %d", d.Attributes.WorldId)
		}
	}

	// Browse world 0 with category=equip => exactly 1 listing.
	resp2, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings?category=equip", srv.URL)))
	if err != nil {
		t.Fatalf("browse filtered: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("browse filtered status = %d, want 200", resp2.StatusCode)
	}
	var env2 struct {
		Data []struct {
			Attributes struct {
				Category string `json:"category"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&env2); err != nil {
		t.Fatalf("decode filtered: %v", err)
	}
	resp2.Body.Close()
	if len(env2.Data) != 1 {
		t.Fatalf("browse(w0, category=equip) returned %d, want 1", len(env2.Data))
	}
	if env2.Data[0].Attributes.Category != "equip" {
		t.Errorf("filtered category = %q, want equip", env2.Data[0].Attributes.Category)
	}

	// Browse world 0 with sellerId=100 => exactly 1 listing (seller 100's own).
	// This backs the channel ENTER_MTS "my sales" announce (GET_USER_SALE_ITEM_DONE).
	resp3, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings?sellerId=100", srv.URL)))
	if err != nil {
		t.Fatalf("browse sellerId: %v", err)
	}
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("browse sellerId status = %d, want 200", resp3.StatusCode)
	}
	var env3 struct {
		Data []struct {
			Attributes struct {
				SellerId uint32 `json:"sellerId"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&env3); err != nil {
		t.Fatalf("decode sellerId: %v", err)
	}
	resp3.Body.Close()
	if len(env3.Data) != 1 {
		t.Fatalf("browse(w0, sellerId=100) returned %d, want 1", len(env3.Data))
	}
	if env3.Data[0].Attributes.SellerId != 100 {
		t.Errorf("filtered sellerId = %d, want 100", env3.Data[0].Attributes.SellerId)
	}
}

// TestListingDetail asserts the detail endpoint returns a single "listings"
// resource and 404s for an unknown id.
func TestListingDetail(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	created := seedListing(t, p, 0, 100, "equip", listing.SaleTypeFixed)

	srv := newListingServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings/%s", srv.URL, created.Id().String())))
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data struct {
			Type string `json:"type"`
			Id   string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	resp.Body.Close()
	if env.Data.Type != "listings" {
		t.Errorf("detail type = %q, want listings", env.Data.Type)
	}
	if env.Data.Id != created.Id().String() {
		t.Errorf("detail id = %q, want %q", env.Data.Id, created.Id().String())
	}

	// Unknown id => 404.
	resp404, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings/%s", srv.URL, "00000000-0000-0000-0000-0000000000ff")))
	if err != nil {
		t.Fatalf("detail 404: %v", err)
	}
	if resp404.StatusCode != http.StatusNotFound {
		t.Errorf("unknown detail status = %d, want 404", resp404.StatusCode)
	}
	resp404.Body.Close()
}

// TestBrowseResponseCarriesTotal asserts the browse response includes the
// JSON:API meta.total (the true match count) and meta.page.last, so a paging
// client renders a real total/last page instead of inferring from the page
// length. Seeds 20 rows, requests a 16-row page, and expects total=20, last=2.
func TestBrowseResponseCarriesTotal(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}
	for i := 0; i < 20; i++ {
		seedListing(t, p, 0, uint32(200+i), "1", listing.SaleTypeFixed)
	}

	srv := newListingServer(t, db)
	defer srv.Close()

	resp, err := (&http.Client{}).Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings?page[number]=1&page[size]=16", srv.URL)))
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("browse status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data []json.RawMessage `json:"data"`
		Meta struct {
			Total int `json:"total"`
			Page  struct {
				Number int `json:"number"`
				Size   int `json:"size"`
				Last   int `json:"last"`
			} `json:"page"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Data) != 16 {
		t.Fatalf("page returned %d rows, want 16", len(env.Data))
	}
	if env.Meta.Total != 20 {
		t.Fatalf("meta.total = %d, want 20 (the full match count, not the page length)", env.Meta.Total)
	}
	if env.Meta.Page.Last != 2 {
		t.Fatalf("meta.page.last = %d, want 2 (ceil(20/16))", env.Meta.Page.Last)
	}
}

// TestBrowseOffsetPreservation pins the invariant this task-117 refactor must
// not break: for a given browse request, the SAME window of rows must come
// back before and after the wire-convention change. Pre-task-117 callers
// windowed with 0-based page/pageSize (page=1, pageSize=16 -> offset 16..29);
// the new 1-based page[number]/page[size] convention must produce the
// identical offset for page[number]=2, page[size]=16 (offset
// (2-1)*16 = 16). Seeds 30 rows so both pages are non-empty, derives the
// "old" window directly from the processor (which still uses the 0-based
// Page/PageSize fields internally), and asserts the new wire request returns
// the exact same set of row ids.
func TestBrowseOffsetPreservation(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}
	for i := 0; i < 30; i++ {
		seedListing(t, p, 0, uint32(700+i), "1", listing.SaleTypeFixed)
	}

	// The pre-task-117 window: page=1 (0-based second page), pageSize=16 ->
	// offset 16..29 (14 rows, since 30 total).
	oldWindow, err := p.Browse(0, listing.StateActive, listing.BrowseFilter{Page: 1, PageSize: 16})
	if err != nil {
		t.Fatalf("old-window Browse: %v", err)
	}
	if len(oldWindow) != 14 {
		t.Fatalf("old window returned %d rows, want 14 (30 total, 16 on page 0)", len(oldWindow))
	}
	oldIds := make(map[string]struct{}, len(oldWindow))
	for _, m := range oldWindow {
		oldIds[m.Id().String()] = struct{}{}
	}

	srv := newListingServer(t, db)
	defer srv.Close()

	// The new wire request: page[number]=2 (1-based second page), page[size]=16.
	resp, err := (&http.Client{}).Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings?page[number]=2&page[size]=16", srv.URL)))
	if err != nil {
		t.Fatalf("browse page[number]=2: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("browse page[number]=2 status = %d, want 200", resp.StatusCode)
	}
	var env struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode page[number]=2: %v", err)
	}
	if len(env.Data) != len(oldIds) {
		t.Fatalf("new page[number]=2 returned %d rows, want %d (same as old page=1)", len(env.Data), len(oldIds))
	}
	for _, d := range env.Data {
		if _, ok := oldIds[d.Id]; !ok {
			t.Errorf("new page[number]=2 row id %q not present in the old page=1 window; offset drifted", d.Id)
		}
	}
}

// TestBrowseInvalidPageParams asserts the endpoint 400s on invalid page
// params (page[size]=0, the legacy bare ?limit=) rather than silently
// clamping or falling back to a default — the repo-wide paginate.ParseParams
// contract (docs/rest-pagination.md).
func TestBrowseInvalidPageParams(t *testing.T) {
	p, db, cleanup := test.CreateListingProcessor(t)
	defer cleanup()
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}
	seedListing(t, p, 0, 900, "1", listing.SaleTypeFixed)

	srv := newListingServer(t, db)
	defer srv.Close()
	client := &http.Client{}

	cases := []string{
		"page[size]=0",
		"limit=10",
	}
	for _, qs := range cases {
		resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/worlds/0/listings?%s", srv.URL, qs)))
		if err != nil {
			t.Fatalf("browse ?%s: %v", qs, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("browse ?%s status = %d, want 400", qs, resp.StatusCode)
		}
	}
}
