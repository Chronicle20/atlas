package job

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"
	"time"

	dataskill "atlas-data/skill"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInfo struct{}

func (t testServerInfo) GetVersion() string { return "1.0.0" }
func (t testServerInfo) GetURI() string     { return "/api/data/" }
func (t testServerInfo) GetPrefix() string  { return "/api/data/" }
func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }

// testDocumentEntity mirrors document.Entity without the PostgreSQL-specific
// column defaults, so it can be AutoMigrated onto sqlite. Copied from
// skill/resource_test.go:28.
type testDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e testDocumentEntity) TableName() string { return "documents" }

func setupResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.New(logrus.StandardLogger(), logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent,
			Colorful:      false,
		}),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&testDocumentEntity{}))
	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func setupTestRouter(db *gorm.DB) *mux.Router {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(db)(testServerInfo{})(router, l)
	return router
}

// seedJob writes a JOB document through the real storage path, so the stored
// `content` is exactly what production writes.
func seedJob(t *testing.T, db *gorm.DB, tenantId uuid.UUID, region string, major, minor uint16, m RestModel) {
	t.Helper()
	tn, err := tenant.Create(tenantId, region, major, minor)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	_, err = NewStorage(l, db).Add(ctx)(m)()
	require.NoError(t, err)
}

func setTenantHeaders(req *http.Request, tenantId uuid.UUID, region string, major, minor uint16) {
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", region)
	req.Header.Set("MAJOR_VERSION", strconv.Itoa(int(major)))
	req.Header.Set("MINOR_VERSION", strconv.Itoa(int(minor)))
}

type singleJobResponse struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Skills []uint32 `json:"skills"`
		} `json:"attributes"`
		Relationships json.RawMessage `json:"relationships"`
	} `json:"data"`
}

func getJobSkills(t *testing.T, db *gorm.DB, path string, tenantId uuid.UUID, region string, major, minor uint16) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	setTenantHeaders(req, tenantId, region, major, minor)
	rr := httptest.NewRecorder()
	setupTestRouter(db).ServeHTTP(rr, req)
	return rr
}

func TestGetJobSkills_Found(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})

	rr := getJobSkills(t, db, "/data/jobs/112/skills", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	var body singleJobResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "jobs", body.Data.Type)
	require.Equal(t, "112", body.Data.Id)
	require.Equal(t, []uint32{1121000, 1121001}, body.Data.Attributes.Skills)
	// PRD §5 pins this response shape as unchanged: no relationships block.
	require.Nil(t, body.Data.Relationships)
}

func TestGetJobSkills_NotFoundWhenAbsentForTenant(t *testing.T) {
	db := setupResourceTestDB(t)
	rr := getJobSkills(t, db, "/data/jobs/99999/skills", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// FR-3.2: "unknown job id" and "job absent from this tenant's version" are
// deliberately the same 404. Job 112 exists for the v95 tenant, not the v61 one.
func TestGetJobSkills_NotFoundForVersionWithoutTheJob(t *testing.T) {
	db := setupResourceTestDB(t)
	newTenant := uuid.New()
	oldTenant := uuid.New()
	seedJob(t, db, newTenant, "GMS", 95, 1, RestModel{Id: 112, Skills: []uint32{1121000}})

	require.Equal(t, http.StatusOK, getJobSkills(t, db, "/data/jobs/112/skills", newTenant, "GMS", 95, 1).Code)
	require.Equal(t, http.StatusNotFound, getJobSkills(t, db, "/data/jobs/112/skills", oldTenant, "GMS", 61, 1).Code)
}

func TestGetJobSkills_TwoTenantsDifferentVersionsDifferentSkills(t *testing.T) {
	db := setupResourceTestDB(t)
	oldTenant := uuid.New()
	newTenant := uuid.New()
	seedJob(t, db, oldTenant, "GMS", 61, 1, RestModel{Id: 510, Skills: []uint32{5101000, 5101001}})
	seedJob(t, db, newTenant, "GMS", 95, 1, RestModel{Id: 510, Skills: []uint32{5101000}})

	var oldBody, newBody singleJobResponse
	rr := getJobSkills(t, db, "/data/jobs/510/skills", oldTenant, "GMS", 61, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &oldBody))

	rr = getJobSkills(t, db, "/data/jobs/510/skills", newTenant, "GMS", 95, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &newBody))

	require.Equal(t, []uint32{5101000, 5101001}, oldBody.Data.Attributes.Skills)
	require.Equal(t, []uint32{5101000}, newBody.Data.Attributes.Skills)
}

func TestGetJobSkills_BadRequest(t *testing.T) {
	db := setupResourceTestDB(t)
	rr := getJobSkills(t, db, "/data/jobs/notanumber/skills", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// seedSkill writes a SKILL document through the real skill storage path, so
// include=skills resolves against production-shaped rows.
func seedSkill(t *testing.T, db *gorm.DB, tenantId uuid.UUID, region string, major, minor uint16, id uint32, name string) {
	t.Helper()
	tn, err := tenant.Create(tenantId, region, major, minor)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	_, err = dataskill.NewStorage(l, db).Add(ctx)(dataskill.RestModel{Id: id, Name: name})()
	require.NoError(t, err)
}

type jobsListResponse struct {
	Data []struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Skills []uint32 `json:"skills"`
		} `json:"attributes"`
		Relationships struct {
			Skills struct {
				Data []struct {
					Type string `json:"type"`
					Id   string `json:"id"`
				} `json:"data"`
			} `json:"skills"`
		} `json:"relationships"`
	} `json:"data"`
	Included []struct {
		Type string `json:"type"`
		Id   string `json:"id"`
	} `json:"included"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

func getJobs(t *testing.T, db *gorm.DB, path string, tenantId uuid.UUID, region string, major, minor uint16) (*httptest.ResponseRecorder, jobsListResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	setTenantHeaders(req, tenantId, region, major, minor)
	rr := httptest.NewRecorder()
	setupTestRouter(db).ServeHTTP(rr, req)
	var body jobsListResponse
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body), "body: %s", rr.Body.String())
	}
	return rr, body
}

func TestGetJobs_ListsTenantJobsWithLinkageAndNoIncluded(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 100, Skills: []uint32{1001000}})
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})

	rr, body := getJobs(t, db, "/data/jobs", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	require.Len(t, body.Data, 2)
	require.Equal(t, 2, body.Meta.Total)
	require.Equal(t, "jobs", body.Data[0].Type)

	// FR-4.3: linkage is always present, derived from the stored id list.
	byId := map[string][]string{}
	for _, d := range body.Data {
		for _, ref := range d.Relationships.Skills.Data {
			require.Equal(t, "skills", ref.Type)
			byId[d.Id] = append(byId[d.Id], ref.Id)
		}
	}
	require.Equal(t, []string{"1001000"}, byId["100"])
	require.Equal(t, []string{"1121000", "1121001"}, byId["112"])

	// NFR §8: no include => no skill drain => no `included` key.
	require.Empty(t, body.Included)
	require.NotContains(t, rr.Body.String(), `"included"`)
}

func TestGetJobs_IncludeSkillsPopulatesIncludedDeduped(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	// 1001000 is shared by both jobs; it must appear once in `included`.
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 100, Skills: []uint32{1001000}})
	seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: 110, Skills: []uint32{1001000, 1101000}})
	seedSkill(t, db, tenantId, "GMS", 83, 1, 1001000, "Power Strike")
	seedSkill(t, db, tenantId, "GMS", 83, 1, 1101000, "Sword Mastery")

	rr, body := getJobs(t, db, "/data/jobs?include=skills", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	ids := make([]string, 0, len(body.Included))
	for _, inc := range body.Included {
		require.Equal(t, "skills", inc.Type)
		ids = append(ids, inc.Id)
	}
	sort.Strings(ids)
	require.Equal(t, []string{"1001000", "1101000"}, ids)
}

func TestGetJobs_EmptyForTenantWithNoJobs(t *testing.T) {
	db := setupResourceTestDB(t)
	rr, body := getJobs(t, db, "/data/jobs", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Empty(t, body.Data)
	require.Equal(t, 0, body.Meta.Total)
}

func TestGetJobs_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	for _, id := range []uint32{100, 110, 111, 112} {
		seedJob(t, db, tenantId, "GMS", 83, 1, RestModel{Id: id, Skills: []uint32{id * 10000}})
	}

	rr, body := getJobs(t, db, "/data/jobs?page[number]=1&page[size]=2", tenantId, "GMS", 83, 1)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, body.Data, 2)
	require.Equal(t, 4, body.Meta.Total)
	require.Equal(t, "100", body.Data[0].Id, "AllPaged orders by document_id")
	require.Equal(t, "110", body.Data[1].Id)
}

func TestGetJobs_RejectsBadPageParams(t *testing.T) {
	db := setupResourceTestDB(t)
	rr, _ := getJobs(t, db, "/data/jobs?page[size]=notanumber", uuid.New(), "GMS", 83, 1)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
