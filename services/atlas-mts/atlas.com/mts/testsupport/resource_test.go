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
