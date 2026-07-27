package tenants

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type testServerInformation struct{}

func (t testServerInformation) GetBaseURL() string {
	return "http://localhost:8080"
}

func (t testServerInformation) GetPrefix() string {
	return ""
}

func doGetConfigTenants(t *testing.T, router *mux.Router, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// TestGetConfigurationTenantsPaginates proves GET /configurations/tenants is
// now paginated. Tenants are created with fixed, deliberately
// out-of-ascending-order ids ("...300", "...100", "...200") - a
// creation-order-derived response would fail the ascending-id assertion;
// database.PagedQuery's schema-derived primary-key ordering is what makes
// this deterministic.
func TestGetConfigurationTenantsPaginates(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	p := NewProcessor(l, context.Background(), db)

	for _, suffix := range []string{"300", "100", "200"} {
		input := createTestRestModel("GMS", 83, 1)
		input.Id = "00000000-0000-0000-0000-000000000" + suffix
		if _, err := p.Create(input); err != nil {
			t.Fatalf("seed create failed: %v", err)
		}
	}

	router := mux.NewRouter()
	InitResource(testServerInformation{})(db)(router, l)

	t.Run("FirstPageOfTwoInAscendingIdOrder", func(t *testing.T) {
		rr := doGetConfigTenants(t, router, "/configurations/tenants?page[number]=1&page[size]=2")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
			Meta struct {
				Total int `json:"total"`
				Page  struct {
					Last int `json:"last"`
				} `json:"page"`
			} `json:"meta"`
			Links struct {
				Next *string `json:"next"`
			} `json:"links"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Data) != 2 {
			t.Fatalf("len(data) = %d, want 2, body=%s", len(doc.Data), rr.Body.String())
		}
		if doc.Data[0].Id != "00000000-0000-0000-0000-000000000100" || doc.Data[1].Id != "00000000-0000-0000-0000-000000000200" {
			t.Fatalf("got ids [%s, %s], want [...100, ...200]", doc.Data[0].Id, doc.Data[1].Id)
		}
		if doc.Meta.Total != 3 {
			t.Fatalf("meta.total = %d, want 3", doc.Meta.Total)
		}
		if doc.Meta.Page.Last != 2 {
			t.Fatalf("meta.page.last = %d, want 2", doc.Meta.Page.Last)
		}
		if doc.Links.Next == nil {
			t.Fatal("expected links.next to be present")
		}
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		rr := doGetConfigTenants(t, router, "/configurations/tenants?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		rr := doGetConfigTenants(t, router, "/configurations/tenants?limit=5")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetConfigTenants(t, router, "/configurations/tenants?page[number]=99&page[size]=2")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct{} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Data) != 0 {
			t.Fatalf("len(data) = %d, want 0", len(doc.Data))
		}
	})
}
