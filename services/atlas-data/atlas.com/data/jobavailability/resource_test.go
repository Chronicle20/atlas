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

// jobAvailabilityResponse mirrors the JSON:API paginated envelope
// (paginate.Envelope.Meta / .BuildLinks): meta.page + top-level links, on
// top of the existing data/attributes shape. Links are plain strings here
// because jsonapi.Link.MarshalJSON emits a bare string whenever Meta is
// empty (data_structs.go), which every link this handler builds is.
type jobAvailabilityResponse struct {
	Data []struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
	Meta struct {
		Total int `json:"total"`
		Page  struct {
			Number int `json:"number"`
			Size   int `json:"size"`
			Last   int `json:"last"`
		} `json:"page"`
	} `json:"meta"`
	Links struct {
		Self  string `json:"self"`
		First string `json:"first"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
		Last  string `json:"last"`
	} `json:"links"`
}

func getJobAvailability(t *testing.T, region string, major, minor uint16) (*httptest.ResponseRecorder, jobAvailabilityResponse) {
	t.Helper()
	return getJobAvailabilityQuery(t, region, major, minor, "")
}

func getJobAvailabilityQuery(t *testing.T, region string, major, minor uint16, rawQuery string) (*httptest.ResponseRecorder, jobAvailabilityResponse) {
	t.Helper()
	path := "/data/job-availability"
	if rawQuery != "" {
		path += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
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

// TestGetJobAvailability_Paginates pins the pagination envelope (task-187
// backend review finding 1): the endpoint MUST paginate per
// docs/rest-pagination.md, not dump the whole in-memory list unbounded.
// It derives the true total from the default (page[size]=50) request --
// large enough to cover every job for a single GMS version -- rather than
// hardcoding a job count, then asserts a small page[size] slices the same
// list and reports a links.next recovery pointer.
func TestGetJobAvailability_Paginates(t *testing.T) {
	full, fullBody := getJobAvailability(t, "GMS", 72, 1)
	require.Equal(t, http.StatusOK, full.Code, "body: %s", full.Body.String())
	total := fullBody.Meta.Total
	require.Greater(t, total, 10, "expected v72 job availability to exceed one small page for this pagination check")
	require.Len(t, fullBody.Data, total, "default page[size]=50 must cover the whole v72 list")
	require.Equal(t, 1, fullBody.Meta.Page.Number)
	require.Equal(t, 50, fullBody.Meta.Page.Size)
	require.Empty(t, fullBody.Links.Next, "the only page must not advertise a next link")

	rr, body := getJobAvailabilityQuery(t, "GMS", 72, 1, "page[size]=10")
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	require.Len(t, body.Data, 10)
	require.Equal(t, total, body.Meta.Total)
	require.Equal(t, 1, body.Meta.Page.Number)
	require.Equal(t, 10, body.Meta.Page.Size)
	require.NotEmpty(t, body.Links.Next, "expected links.next since more than one page remains")
}
