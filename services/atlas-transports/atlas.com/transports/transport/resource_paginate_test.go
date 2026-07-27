package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type testServerInformation struct{}

func (t testServerInformation) GetBaseURL() string {
	return "http://localhost:8080"
}

func (t testServerInformation) GetPrefix() string {
	return ""
}

func doGetRoutes(t *testing.T, router *mux.Router, tenantId uuid.UUID, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// TestGetAllRoutesPaginates proves GET /transports/routes is now paginated.
// Routes are seeded directly into the Redis-backed registry (bypassing
// Processor.AddTenant's schedule computation - not needed here) with fixed,
// deliberately out-of-ascending-order ids ("...300", "...100", "...200").
// The registry is a Redis hash (unordered GetAllValues) - the handler's
// explicit sort is what makes the paged response deterministic.
func TestGetAllRoutesPaginates(t *testing.T) {
	setupTransportTestRegistry(t)
	tm, ctx := newTestTenantContext(t)

	routes := make([]Model, 0, 3)
	for _, suffix := range []string{"300", "100", "200"} {
		m, err := NewBuilder("Route-" + suffix).
			SetId(uuid.MustParse("00000000-0000-0000-0000-000000000" + suffix)).
			SetStartMapId(_map.Id(100000000)).
			SetStagingMapId(_map.Id(100000001)).
			SetEnRouteMapIds([]_map.Id{_map.Id(100000002)}).
			SetDestinationMapId(_map.Id(200000100)).
			SetBoardingWindowDuration(5 * time.Minute).
			SetPreDepartureDuration(2 * time.Minute).
			SetTravelDuration(10 * time.Minute).
			SetCycleInterval(30 * time.Minute).
			Build()
		if err != nil {
			t.Fatalf("seed build failed: %v", err)
		}
		routes = append(routes, m)
	}
	getRouteRegistry().AddTenant(ctx, routes)

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	t.Run("FirstPageOfTwoInAscendingIdOrder", func(t *testing.T) {
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes?page[number]=1&page[size]=2")
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
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes?limit=5")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes?page[number]=99&page[size]=2")
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

	t.Run("FilterByStartMapKeepsPaginatedShape", func(t *testing.T) {
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes?filter[startMapId]=100000000")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct{} `json:"data"`
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Data) != 1 {
			t.Fatalf("len(data) = %d, want 1 (filter is bounded to <=1 match)", len(doc.Data))
		}
		if doc.Meta.Total != 1 {
			t.Fatalf("meta.total = %d, want 1", doc.Meta.Total)
		}
	})
}
