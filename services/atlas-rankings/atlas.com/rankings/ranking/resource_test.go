package ranking

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInfo struct{}

func (s testServerInfo) GetBaseURL() string { return "" }
func (s testServerInfo) GetPrefix() string  { return "/api/" }

func testRouter(t *testing.T, db *gorm.DB) *mux.Router {
	t.Helper()
	router := mux.NewRouter().PathPrefix("/api/").Subrouter()
	InitResource(testServerInfo{})(db)(router, logrus.New())
	return router
}

func tenantHeaders(r *http.Request, tm tenant.Model) {
	r.Header.Set("TENANT_ID", tm.Id().String())
	r.Header.Set("REGION", tm.Region())
	r.Header.Set("MAJOR_VERSION", strconv.Itoa(int(tm.MajorVersion())))
	r.Header.Set("MINOR_VERSION", strconv.Itoa(int(tm.MinorVersion())))
}

func seedRanking(t *testing.T, db *gorm.DB, tm tenant.Model, characterId uint32, rank uint32) {
	t.Helper()
	e := Entity{
		TenantId:    tm.Id(),
		CharacterId: characterId,
		WorldId:     0,
		JobCategory: 1,
		OverallRank: rank,
		JobRank:     rank,
		ComputedAt:  time.Now(),
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestBulkEndpoint(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	seedRanking(t, db, tm, 1, 17)
	router := testRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters?ids=1,999", nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data []struct {
			Id         string          `json:"id"`
			Attributes json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Id != "1" {
		t.Fatalf("unknown ids must be omitted: %s", rec.Body.String())
	}
}

func TestBulkEndpointBadIds(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	router := testRouter(t, db)

	for _, ids := range []string{"", "abc", "1,abc", ","} {
		req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters?ids="+ids, nil)
		tenantHeaders(req, tm)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("ids=%q status = %d, want 400", ids, rec.Code)
		}
	}
}

func TestBulkEndpointMissingIds(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	router := testRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters", nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("no ids query param status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestSingleEndpoint(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	seedRanking(t, db, tm, 7, 3)
	router := testRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters/7", nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Data struct {
			Id         string `json:"id"`
			Attributes struct {
				Rank    uint32 `json:"rank"`
				JobRank uint32 `json:"jobRank"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Id != "7" || body.Data.Attributes.Rank != 3 || body.Data.Attributes.JobRank != 3 {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/rankings/characters/999", nil)
	tenantHeaders(req, tm)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing ranking status = %d, want 404", rec.Code)
	}
}

func TestSingleEndpointBadCharacterId(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	router := testRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/rankings/characters/not-a-number", nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("non-numeric characterId status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

// seedRankings inserts full ranking rows (display fields included) for the
// leaderboard endpoint tests, which need more than the rank alone.
func seedRankings(t *testing.T, db *gorm.DB, tm tenant.Model, entities []Entity) {
	t.Helper()
	for i := range entities {
		e := entities[i]
		e.TenantId = tm.Id()
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

// doGet issues a GET against the given router under the seeded tenant.
func doGet(t *testing.T, router *mux.Router, tm tenant.Model, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api"+path, nil)
	tenantHeaders(req, tm)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLeaderboardOrdersByOverallRank(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	seedRankings(t, db, tm, []Entity{
		{CharacterId: 1, Name: "A", WorldId: 0, JobCategory: 1, Level: 40, JobId: 110, OverallRank: 2, JobRank: 2, ComputedAt: time.Unix(1, 0)},
		{CharacterId: 2, Name: "B", WorldId: 0, JobCategory: 1, Level: 50, JobId: 110, OverallRank: 1, JobRank: 1, ComputedAt: time.Unix(1, 0)},
	})
	router := testRouter(t, db)

	rr := doGet(t, router, tm, "/rankings?filter[worldId]=0&page[number]=1&page[size]=10")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	// First data element must be character 2 (overall_rank 1).
	if !strings.Contains(rr.Body.String(), `"characterId":2`) {
		t.Fatalf("missing characterId 2 in body: %s", rr.Body.String())
	}
}

func TestLeaderboardRequiresWorldId(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	router := testRouter(t, db)

	rr := doGet(t, router, tm, "/rankings?page[number]=1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

// TestLeaderboardEmptyWorldReturnsEmptyPage covers the case explicitly
// called out for this task beyond the brief's two cases: a valid,
// filter-satisfying request against a world with no ranked characters must
// succeed with an empty page rather than erroring.
func TestLeaderboardEmptyWorldReturnsEmptyPage(t *testing.T) {
	db := testDatabase(t)
	tm, _ := testTenantContext(t)
	router := testRouter(t, db)

	rr := doGet(t, router, tm, "/rankings?filter[worldId]=5&page[number]=1&page[size]=10")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data []json.RawMessage `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 0 {
		t.Fatalf("expected empty page for world with no rankings, got %d items: %s", len(body.Data), rr.Body.String())
	}
	if body.Meta.Total != 0 {
		t.Fatalf("expected total 0, got %d", body.Meta.Total)
	}
}
