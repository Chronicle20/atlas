package ranking

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
