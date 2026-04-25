package job

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type testServerInfo struct{}

func (t testServerInfo) GetVersion() string { return "1.0.0" }
func (t testServerInfo) GetURI() string     { return "/api/data/" }
func (t testServerInfo) GetPrefix() string  { return "/api/data/" }
func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }

func setRequiredTenantHeaders(req *http.Request) {
	req.Header.Set("TENANT_ID", uuid.New().String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
}

type apiResponse struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Skills []uint32 `json:"skills"`
		} `json:"attributes"`
	} `json:"data"`
}

func TestGetJobSkills_Found(t *testing.T) {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	si := testServerInfo{}
	InitResource(si)(router, l)

	req := httptest.NewRequest(http.MethodGet, "/data/jobs/112/skills", nil)
	setRequiredTenantHeaders(req)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	var body apiResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "jobs", body.Data.Type)
	require.Equal(t, "112", body.Data.Id)
	require.NotEmpty(t, body.Data.Attributes.Skills)
}

func TestGetJobSkills_NotFound(t *testing.T) {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	si := testServerInfo{}
	InitResource(si)(router, l)

	req := httptest.NewRequest(http.MethodGet, "/data/jobs/99999/skills", nil)
	setRequiredTenantHeaders(req)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetJobSkills_BadRequest(t *testing.T) {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	si := testServerInfo{}
	InitResource(si)(router, l)

	req := httptest.NewRequest(http.MethodGet, "/data/jobs/notanumber/skills", nil)
	setRequiredTenantHeaders(req)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}
