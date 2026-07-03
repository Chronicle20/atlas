package quest

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

// TestQuestsBareList_PaginationEnvelope exercises the task-117 recipe on
// GET /data/quests: page[size]=2 over 3 seeded quests returns page 1 (2
// items), a total of 3, and a next link.
func TestQuestsBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestQuestData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/quests?page[number]=1&page[size]=2", ts.URL)
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
	page := doc.Meta["page"].(map[string]interface{})
	assert.EqualValues(t, 1, page["number"])
	assert.EqualValues(t, 2, page["size"])
	assert.EqualValues(t, 2, page["last"])
	assert.NotNil(t, doc.Links["next"])
}

// TestQuestsBareList_RejectsBadPageSize confirms invalid page[size] is a 400,
// not a silently clamped/ignored parameter.
func TestQuestsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestQuestData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/quests?page[size]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestQuestsAutoStart_PaginationEnvelope covers the derived-filter by-parent
// style route: only quests 1000 and 1002 have AutoStart=true in the fixture
// (see setupTestQuestData), so page[size]=1 must report total=2 with a next
// link, proving the filter runs before Slice.
func TestQuestsAutoStart_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestQuestData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/quests/auto-start?page[number]=1&page[size]=1", ts.URL)
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

	assert.Len(t, doc.Data, 1)
	assert.EqualValues(t, 2, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

// TestQuestsAutoStart_RejectsBadPageNumber confirms the by-parent-style
// derived-filter route shares the same 400 contract as the bare route.
func TestQuestsAutoStart_RejectsBadPageNumber(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestQuestData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/quests/auto-start?page[number]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
