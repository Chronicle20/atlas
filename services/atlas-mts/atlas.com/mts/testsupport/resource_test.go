package testsupport_test

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"atlas-mts/testsupport"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type testServerInfo struct{}

func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t testServerInfo) GetPrefix() string  { return "/api" }

func newTestServer(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	testsupport.InitResource(testServerInfo{})(db)(router, l)
	return httptest.NewServer(router)
}

func withTenant(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("TENANT_ID", test.TestTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedBody(t *testing.T, attributes map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"data": map[string]any{"type": "test-seeds", "attributes": attributes},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return b
}

// TestSeedListings asserts a mixed seed call creates active rows with real
// serials and derived categories, and that they surface in a normal browse.
func TestSeedListings(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{
			{"saleType": "fixed", "count": 3, "templateId": 1302000, "listValue": 1000},
			{"saleType": "auction", "count": 2, "templateId": 2000000, "quantity": 50, "listValue": 500, "durationSeconds": 30},
		},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var envelope struct {
		Data []struct {
			Id         string `json:"id"`
			Attributes struct {
				ItcSn    uint32  `json:"itcSn"`
				SaleType string  `json:"saleType"`
				State    string  `json:"state"`
				Category string  `json:"category"`
				EndsAt   *string `json:"endsAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) != 5 {
		t.Fatalf("expected 5 seeded listings, got %d", len(envelope.Data))
	}
	serials := map[uint32]bool{}
	auctions := 0
	for _, d := range envelope.Data {
		if d.Attributes.State != "active" {
			t.Fatalf("expected active, got %s", d.Attributes.State)
		}
		if d.Attributes.ItcSn == 0 || serials[d.Attributes.ItcSn] {
			t.Fatalf("expected distinct non-zero serials, got %d twice or zero", d.Attributes.ItcSn)
		}
		serials[d.Attributes.ItcSn] = true
		switch d.Attributes.SaleType {
		case "auction":
			auctions++
			if d.Attributes.Category != "3" {
				t.Fatalf("auction category = %q, want \"3\"", d.Attributes.Category)
			}
			if d.Attributes.EndsAt == nil {
				t.Fatal("auction missing endsAt")
			}
		case "fixed":
			if d.Attributes.Category != "1" {
				t.Fatalf("fixed category = %q, want \"1\"", d.Attributes.Category)
			}
		default:
			t.Fatalf("unexpected saleType %q", d.Attributes.SaleType)
		}
	}
	if auctions != 2 {
		t.Fatalf("expected 2 auctions, got %d", auctions)
	}

	// Seeded rows are real: the production browse sees all 5.
	ms, err := listing.NewProcessor(logrus.New(), test.CreateTestContext(), db).Browse(0, listing.StateActive, listing.BrowseFilter{})
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if len(ms) != 5 {
		t.Fatalf("browse found %d listings, want 5", len(ms))
	}
}

// TestSeedCap asserts the 200-listing cap returns 400.
func TestSeedCap(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{{"saleType": "fixed", "count": 201, "templateId": 1302000, "listValue": 100}},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

// TestSeedCapBoundary asserts the cap is inclusive: exactly 200 succeeds,
// 201 still fails. Uses a single entry with count 200 to keep it fast.
func TestSeedCapBoundary(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{{"saleType": "fixed", "count": 200, "templateId": 1302000, "listValue": 100}},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for exactly-200, got %d", res.StatusCode)
	}

	var envelope struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) != 200 {
		t.Fatalf("expected 200 seeded listings, got %d", len(envelope.Data))
	}
}

// TestSeedListingsDefaults asserts an entry supplying only saleType and
// templateId lands with the documented defaults: sellerName "TestSeller",
// sellerId 999000001, listValue 1000, quantity 1, endsAt ~= now+300s, and
// creates exactly 1 listing.
func TestSeedListingsDefaults(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	before := time.Now()
	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{
			{"saleType": "auction", "templateId": 1302000},
		},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var envelope struct {
		Data []struct {
			Id         string `json:"id"`
			Attributes struct {
				SellerId   uint32  `json:"sellerId"`
				SellerName string  `json:"sellerName"`
				ListValue  uint32  `json:"listValue"`
				Quantity   uint32  `json:"quantity"`
				EndsAt     *string `json:"endsAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) != 1 {
		t.Fatalf("expected exactly 1 seeded listing, got %d", len(envelope.Data))
	}

	a := envelope.Data[0].Attributes
	if a.SellerId != 999000001 {
		t.Fatalf("sellerId default = %d, want 999000001", a.SellerId)
	}
	if a.SellerName != "TestSeller" {
		t.Fatalf("sellerName default = %q, want %q", a.SellerName, "TestSeller")
	}
	if a.ListValue != 1000 {
		t.Fatalf("listValue default = %d, want 1000", a.ListValue)
	}
	if a.Quantity != 1 {
		t.Fatalf("quantity default = %d, want 1", a.Quantity)
	}
	if a.EndsAt == nil {
		t.Fatal("expected endsAt to be set for auction")
	}
	endsAt, err := time.Parse(time.RFC3339, *a.EndsAt)
	if err != nil {
		t.Fatalf("parse endsAt %q: %v", *a.EndsAt, err)
	}
	min := before.Add(4 * time.Minute)
	max := before.Add(6 * time.Minute)
	if endsAt.Before(min) || endsAt.After(max) {
		t.Fatalf("endsAt = %v, want between %v and %v (now+300s default)", endsAt, min, max)
	}

	// Confirm exactly 1 row landed and the seller default persisted to the DB.
	m, err := listing.GetById(envelope.Data[0].Id)(db)()
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.SellerId() != 999000001 {
		t.Fatalf("stored sellerId = %d, want 999000001", m.SellerId())
	}
}
