package instance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
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

func doGetInstance(t *testing.T, router *mux.Router, tenantId uuid.UUID, path string) *httptest.ResponseRecorder {
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

// TestGetAllInstanceRoutesPaginates proves GET /transports/instance-routes
// is now paginated. Routes are seeded directly into the Redis-backed
// registry with fixed, deliberately out-of-ascending-order ids
// ("...300", "...100", "...200") - the registry is a Redis hash (unordered
// GetAllValues); the handler's explicit sort is what makes the paged
// response deterministic.
func TestGetAllInstanceRoutesPaginates(t *testing.T) {
	setupRouteTestRegistry(t)
	tenantId := uuid.New()
	tm, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("seed tenant create failed: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)

	for _, suffix := range []string{"300", "100", "200"} {
		route, buildErr := NewRouteBuilder("Route-" + suffix).
			SetId(uuid.MustParse("00000000-0000-0000-0000-000000000" + suffix)).
			SetStartMapId(_map.Id(100000000)).
			SetTransitMapIds([]_map.Id{100000100}).
			SetDestinationMapId(_map.Id(100000200)).
			SetCapacity(3).
			SetBoardingWindow(10 * time.Second).
			SetTravelDuration(30 * time.Second).
			Build()
		if buildErr != nil {
			t.Fatalf("seed build failed: %v", buildErr)
		}
		getRouteRegistry().AddTenant(ctx, []RouteModel{route})
	}

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	t.Run("FirstPageOfTwoInAscendingIdOrder", func(t *testing.T) {
		rr := doGetInstance(t, router, tenantId, "/transports/instance-routes?page[number]=1&page[size]=2")
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
		rr := doGetInstance(t, router, tenantId, "/transports/instance-routes?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		rr := doGetInstance(t, router, tenantId, "/transports/instance-routes?limit=5")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetInstance(t, router, tenantId, "/transports/instance-routes?page[number]=99&page[size]=2")
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

// TestGetInstanceRouteStatusPaginates proves
// GET /transports/instance-routes/{routeId}/status is now paginated.
// GetInstanceRouteStatusHandler hardcodes uuid.Nil as the tenant id passed
// to InstanceRegistry.GetInstancesByRoute (a pre-existing quirk, unrelated
// to this task and left untouched) - instances are seeded under that same
// uuid.Nil tenant so the handler can find them. The route's capacity is 1
// and each instance is filled to capacity before the next
// FindOrCreateInstance call, forcing 3 distinct instances instead of one
// reused instance. Instance ids are server-generated (uuid.New()), so the
// determinism assertion sorts the expected ids the same way the handler
// does rather than asserting fixed literals.
func TestGetInstanceRouteStatusPaginates(t *testing.T) {
	setupInstanceTestRegistry(t)

	route, err := NewRouteBuilder("status-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(1).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()
	if err != nil {
		t.Fatalf("seed build failed: %v", err)
	}

	reg := getInstanceRegistry()
	now := time.Now()
	seededIds := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		inst := reg.FindOrCreateInstance(uuid.Nil, route, now)
		reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: uint32(i + 1)})
		seededIds = append(seededIds, inst.InstanceId().String())
	}

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	path := "/transports/instance-routes/" + route.Id().String() + "/status"

	t.Run("FirstPageOfTwoInAscendingIdOrder", func(t *testing.T) {
		rr := doGetInstance(t, router, uuid.New(), path+"?page[number]=1&page[size]=2")
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
		if doc.Data[0].Id >= doc.Data[1].Id {
			t.Fatalf("page 1 not in ascending id order: %s, %s", doc.Data[0].Id, doc.Data[1].Id)
		}
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		rr := doGetInstance(t, router, uuid.New(), path+"?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetInstance(t, router, uuid.New(), path+"?page[number]=99&page[size]=2")
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
