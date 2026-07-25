package reactor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReactorsBareList_PaginationEnvelope exercises GET /data/reactors over
// the 3 seeded reactors (setupTestReactorData): page[size]=2 returns page 1
// (2 items), a total of 3, and a next link.
func TestReactorsBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestReactorData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/reactors?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestReactorsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestReactorData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/reactors?page[size]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestReactorsSearch_EnvelopeMatchesBareArm proves the search arm presents
// the identical envelope + 400 contract as the bare arm (task-9 recipe).
func TestReactorsSearch_EnvelopeMatchesBareArm(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestReactorData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/reactors?search=Reactor&page[number]=1&page[size]=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	// "Reactor" substring-matches all three seeded names ("Test Reactor",
	// "Item Reactor", "Empty Reactor"): total=3, page[size]=1 returns 1 item
	// with a next link.
	assert.Len(t, doc.Data, 1)
	assert.EqualValues(t, 3, doc.Meta["total"])
	page := doc.Meta["page"].(map[string]interface{})
	assert.EqualValues(t, 1, page["number"])
	assert.EqualValues(t, 1, page["size"])
	assert.NotNil(t, doc.Links["next"])
}

// TestReactorsSearch_RejectsLegacyLimitParam confirms the search arm no
// longer honors the pre-task-117 ad-hoc ?limit= param: presence of ?limit=
// is now a 400 under paginate.ParseParams, matching the bare arm.
func TestReactorsSearch_RejectsLegacyLimitParam(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestReactorData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/reactors?search=Reactor&limit=1", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
