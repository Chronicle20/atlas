package listing_test

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
