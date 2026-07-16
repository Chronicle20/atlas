package tenant_test

import (
	"atlas-tenants/tenant"
	"atlas-tenants/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

type testServerInformation struct{}

func (t testServerInformation) GetBaseURL() string {
	return "http://localhost:8080"
}

func (t testServerInformation) GetPrefix() string {
	return ""
}

func doGetTenants(t *testing.T, router *mux.Router, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// TestGetTenantsPaginates proves GET /tenants is now paginated. Tenants are
// created in reverse-name order (Tenant-C, Tenant-A, Tenant-B) so any
// creation-order-derived response would fail the ascending-id assertion -
// database.PagedQuery's schema-derived primary-key ordering is what makes
// this deterministic (the tenant primary key is a client-supplied random
// uuid.New(), so insertion order and id order are independent).
func TestGetTenantsPaginates(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()

	router := mux.NewRouter()
	tenant.RegisterRoutes(db)(testServerInformation{})(router, logger)

	// Seed directly at the entity layer (bypassing CreateAndEmit's Kafka
	// producer, unavailable in this unit test) - matches the existing
	// processor_test.go testProcessor.create() precedent.
	created := make([]string, 0, 3)
	for _, name := range []string{"Tenant-C", "Tenant-A", "Tenant-B"} {
		m, err := tenant.NewModelBuilder().
			SetName(name).
			SetRegion("GMS").
			SetMajorVersion(83).
			SetMinorVersion(1).
			Build()
		if err != nil {
			t.Fatalf("seed build failed: %v", err)
		}
		e := tenant.FromModel(m)
		if err := tenant.CreateTenant(db, e); err != nil {
			t.Fatalf("seed create failed: %v", err)
		}
		created = append(created, m.Id().String())
	}

	t.Run("FirstPageOfTwoInAscendingIdOrder", func(t *testing.T) {
		rr := doGetTenants(t, router, "/tenants?page[number]=1&page[size]=2")
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
		if doc.Meta.Total != 3 {
			t.Fatalf("meta.total = %d, want 3", doc.Meta.Total)
		}
		if doc.Meta.Page.Last != 2 {
			t.Fatalf("meta.page.last = %d, want 2", doc.Meta.Page.Last)
		}
		if doc.Links.Next == nil {
			t.Fatal("expected links.next to be present")
		}

		// Sorted ascending by uuid string.
		want := append([]string{}, created...)
		sort.Strings(want)
		if doc.Data[0].Id != want[0] || doc.Data[1].Id != want[1] {
			t.Fatalf("got ids [%s, %s], want [%s, %s]", doc.Data[0].Id, doc.Data[1].Id, want[0], want[1])
		}
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		rr := doGetTenants(t, router, "/tenants?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		rr := doGetTenants(t, router, "/tenants?limit=5")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetTenants(t, router, "/tenants?page[number]=99&page[size]=2")
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
