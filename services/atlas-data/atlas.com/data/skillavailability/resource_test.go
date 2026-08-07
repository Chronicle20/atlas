package skillavailability

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

// skillAvailabilityResponse mirrors the JSON:API paginated envelope
// (paginate.Envelope.Meta / .BuildLinks): meta.page + top-level links, on
// top of the data/attributes shape. Links are plain strings because
// jsonapi.Link.MarshalJSON emits a bare string whenever Meta is empty
// (api2go/jsonapi/data_structs.go), which every link this handler builds is.
type skillAvailabilityResponse struct {
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

func getSkillAvailability(t *testing.T, region string, major, minor uint16) (*httptest.ResponseRecorder, skillAvailabilityResponse) {
	t.Helper()
	return getSkillAvailabilityQuery(t, region, major, minor, "")
}

func getSkillAvailabilityQuery(t *testing.T, region string, major, minor uint16, rawQuery string) (*httptest.ResponseRecorder, skillAvailabilityResponse) {
	t.Helper()
	path := "/data/skill-availability"
	if rawQuery != "" {
		path += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	setTenantHeaders(req, region, major, minor)
	rr := httptest.NewRecorder()
	setupTestRouter().ServeHTTP(rr, req)
	var body skillAvailabilityResponse
	if rr.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body), "body: %s", rr.Body.String())
	}
	return rr, body
}

// TestGetSkillAvailability_ShapeAndType is an HTTP-level smoke test: the
// endpoint responds 200, resource type is "skill-availability", and a
// version-stable low-wire-id skill (BeginnerThreeSnails, wire 1000 -- see
// processor_test.go for the identity-level pin) round-trips through the
// handler within the first page.
func TestGetSkillAvailability_ShapeAndType(t *testing.T) {
	rr, body := getSkillAvailability(t, "GMS", 48, 1)
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	require.NotEmpty(t, body.Data)

	found := false
	for _, d := range body.Data {
		require.Equal(t, "skill-availability", d.Type)
		if d.Id == "1000" {
			found = true
			require.Equal(t, "Beginner Three Snails", d.Attributes.Name)
		}
	}
	require.True(t, found, "expected wire id 1000 (Beginner Three Snails) within the first page")
}

// TestGetSkillAvailability_Paginates pins the pagination envelope
// (task-187 backend review finding 1): the endpoint MUST paginate per
// docs/rest-pagination.md, not dump the whole in-memory list unbounded. It
// derives the true total from the default (page[size]=50) request rather
// than hardcoding a skill count -- the skill list is far larger than the
// job list, so a small page[size] is guaranteed to produce a links.next.
func TestGetSkillAvailability_Paginates(t *testing.T) {
	full, fullBody := getSkillAvailability(t, "GMS", 72, 1)
	require.Equal(t, http.StatusOK, full.Code, "body: %s", full.Body.String())
	total := fullBody.Meta.Total
	require.Greater(t, total, 50, "expected v72 skill availability to exceed one default page for this pagination check")
	require.Len(t, fullBody.Data, 50, "default page[size]=50 must be capped, not the whole list")
	require.Equal(t, 1, fullBody.Meta.Page.Number)
	require.Equal(t, 50, fullBody.Meta.Page.Size)
	require.NotEmpty(t, fullBody.Links.Next, "expected links.next since more than one page remains")

	rr, body := getSkillAvailabilityQuery(t, "GMS", 72, 1, "page[size]=10")
	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	require.Len(t, body.Data, 10)
	require.Equal(t, total, body.Meta.Total)
	require.Equal(t, 1, body.Meta.Page.Number)
	require.Equal(t, 10, body.Meta.Page.Size)
	require.NotEmpty(t, body.Links.Next, "expected links.next since more than one page remains")
}
