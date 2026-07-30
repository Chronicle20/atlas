package jobavailability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func setupTestRouter() *mux.Router {
	router := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(testServerInfo{})(router, l)
	return router
}

func setTenantHeaders(req *http.Request, region string, major, minor uint16) {
	req.Header.Set("TENANT_ID", uuid.New().String())
	req.Header.Set("REGION", region)
	req.Header.Set("MAJOR_VERSION", strconv.Itoa(int(major)))
	req.Header.Set("MINOR_VERSION", strconv.Itoa(int(minor)))
}

type jobAvailabilityResponse struct {
	Data []struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

func getJobAvailability(t *testing.T, region string, major, minor uint16) (*httptest.ResponseRecorder, jobAvailabilityResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/data/job-availability", nil)
	setTenantHeaders(req, region, major, minor)
	rr := httptest.NewRecorder()
	setupTestRouter().ServeHTTP(rr, req)
	var body jobAvailabilityResponse
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body), "body: %s", rr.Body.String())
	}
	return rr, body
}

// TestGetJobAvailability_V48HasGmNotPirate pins the pre-Pirate wire binding:
// at GMS 48.1, wire id 500 is Gm ("Gm" -- Set.Name's exact casing, not
// "GM"), and no Pirate entry exists at all (Pirate wasn't released yet).
func TestGetJobAvailability_V48HasGmNotPirate(t *testing.T) {
	rr, body := getJobAvailability(t, "GMS", 48, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	found := false
	for _, d := range body.Data {
		require.Equal(t, "job-availability", d.Type)
		require.NotEqual(t, "Pirate", d.Attributes.Name, "v48 must not include a released Pirate")
		if d.Id == "500" {
			found = true
			require.Equal(t, "Gm", d.Attributes.Name)
		}
	}
	require.True(t, found, "expected wire id 500 (Gm) in v48 availability list")
}

// TestGetJobAvailability_V72HasPirate pins the post-Pirate wire binding: at
// GMS 72.1, wire id 500 is Pirate (Gm moved to 900).
func TestGetJobAvailability_V72HasPirate(t *testing.T) {
	rr, body := getJobAvailability(t, "GMS", 72, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	found := false
	for _, d := range body.Data {
		if d.Id == "500" {
			found = true
			require.Equal(t, "Pirate", d.Attributes.Name)
		}
	}
	require.True(t, found, "expected wire id 500 (Pirate) in v72 availability list")
}
