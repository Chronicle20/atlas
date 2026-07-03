package quest

import (
	"atlas-data/document"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

// seedScrambledAutoStartQuests seeds 5 auto-start quests (ids 5000-5004)
// PLUS 2 non-auto-start quests (5010, 5011), inserted in an order that does
// NOT match ascending quest id. This exercises the task-117 fix: the
// auto-start handler filters an unordered document.Storage.DrainAllProvider()
// result (which internally pages a plain "type = ?" Find with no ORDER BY —
// see document/db_storage.go DbStorage.All) before paginate.Slice. Without a
// stable sort keyed on the quest id, the page split reflects whatever order
// the DB/registry happens to return rather than a deterministic total order.
func seedScrambledAutoStartQuests(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	storage := document.NewStorage(l, db, GetModelRegistry(), "QUEST")

	// Insertion order deliberately scrambled relative to ascending id, and
	// interleaved with non-auto-start quests, so a naive unsorted GetAll
	// would neither be sorted by id nor contiguous for the filtered set.
	quests := []RestModel{
		{Id: 5003, Name: "Auto 3", AutoStart: true},
		{Id: 5010, Name: "Manual 1", AutoStart: false},
		{Id: 5000, Name: "Auto 0", AutoStart: true},
		{Id: 5004, Name: "Auto 4", AutoStart: true},
		{Id: 5011, Name: "Manual 2", AutoStart: false},
		{Id: 5001, Name: "Auto 1", AutoStart: true},
		{Id: 5002, Name: "Auto 2", AutoStart: true},
	}
	for _, q := range quests {
		_, err := storage.Add(ctx)(q)()
		require.NoError(t, err)
	}
}

// TestQuestsAutoStart_CrossPageDeterminism pins the task-117 fix: fetching
// every page of GET /data/quests/auto-start (page[size]=2) over 5 seeded
// auto-start quests must return the exact seeded set with no duplicates, no
// drops, and quest-id-ordered page boundaries (page 1 = the two lowest ids,
// etc.) — matching the stable-sort-by-id contract already used by
// npc/{id}/quests. Without the sort.SliceStable added to
// handleGetAutoStartQuests, this assertion is only accidentally true when
// the DB returns rows in an order that happens to match ascending id; the
// scrambled insertion order here is chosen so that a regression (removing
// the sort) is a real behavioral difference, not just a theoretical one.
func TestQuestsAutoStart_CrossPageDeterminism(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	seedScrambledAutoStartQuests(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	type page struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}

	fetchPage := func(number int) page {
		url := fmt.Sprintf("%s/data/quests/auto-start?page[number]=%d&page[size]=2", ts.URL, number)
		resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var p page
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&p))
		return p
	}

	p1 := fetchPage(1)
	p2 := fetchPage(2)
	p3 := fetchPage(3)

	require.Len(t, p1.Data, 2)
	require.Len(t, p2.Data, 2)
	require.Len(t, p3.Data, 1)
	assert.EqualValues(t, 5, p1.Meta["total"])
	assert.EqualValues(t, 5, p2.Meta["total"])
	assert.EqualValues(t, 5, p3.Meta["total"])

	// Page boundaries are deterministic and ordered by quest id: no overlap,
	// no gap, exactly the seeded auto-start set.
	assert.Equal(t, "5000", p1.Data[0].Id)
	assert.Equal(t, "5001", p1.Data[1].Id)
	assert.Equal(t, "5002", p2.Data[0].Id)
	assert.Equal(t, "5003", p2.Data[1].Id)
	assert.Equal(t, "5004", p3.Data[0].Id)

	seen := make(map[string]int)
	for _, d := range append(append(p1.Data, p2.Data...), p3.Data...) {
		seen[d.Id]++
	}
	assert.Len(t, seen, 5, "expected exactly 5 distinct auto-start quests across all pages")
	for id, count := range seen {
		assert.Equal(t, 1, count, "quest %s appeared on more than one page", id)
	}
	for _, wantId := range []string{"5000", "5001", "5002", "5003", "5004"} {
		assert.Contains(t, seen, wantId)
	}
	// Non-auto-start quests must never leak into the filtered result.
	assert.NotContains(t, seen, "5010")
	assert.NotContains(t, seen, "5011")
}
