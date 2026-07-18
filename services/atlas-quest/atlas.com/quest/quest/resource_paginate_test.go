package quest

import (
	"atlas-quest/quest/progress"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type paginateTestServerInformation struct{}

func (t *paginateTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *paginateTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &paginateTestServerInformation{}

func setupQuestRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&paginateTestServerInformation{})(db)
	ri(r, l)
	return r
}

func requestQuestsWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedQuest(t *testing.T, db *gorm.DB, tenantId uuid.UUID, characterId uint32, questId uint32, state State) Entity {
	t.Helper()
	e := Entity{
		TenantId:    tenantId,
		CharacterId: characterId,
		QuestId:     questId,
		State:       state,
		StartedAt:   time.Now(),
	}
	require.NoError(t, db.Create(&e).Error)
	return e
}

func pinSingleConnection(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
}

// TestGetQuestsByCharacterPaginates drives GET
// /characters/{characterId}/quests through the real resource router,
// verifying the JSON:API paginated envelope, 400 on invalid paging params,
// and empty-page handling past the last page. Also confirms another
// character's quests are excluded from the total.
func TestGetQuestsByCharacterPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, progress.Migration)
	pinSingleConnection(t, db)
	tenantId := uuid.New()

	seedQuest(t, db, tenantId, 1, 100, StateStarted)
	seedQuest(t, db, tenantId, 1, 101, StateStarted)
	seedQuest(t, db, tenantId, 1, 102, StateCompleted)
	seedQuest(t, db, tenantId, 2, 200, StateStarted)

	srv := httptest.NewServer(setupQuestRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/quests?page[number]=1&page[size]=2", srv.URL)
		req := requestQuestsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude character 2's quest")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/quests?page[size]=0", srv.URL)
		req := requestQuestsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/quests?limit=5", srv.URL)
		req := requestQuestsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/quests?page[number]=99&page[size]=2", srv.URL)
		req := requestQuestsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
		assert.NotContains(t, doc.Links, "next")
	})
}

// TestGetStartedQuestsByCharacterPaginates drives GET
// /characters/{characterId}/quests/started, confirming the state filter is
// applied AND the response is paginated.
func TestGetStartedQuestsByCharacterPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, progress.Migration)
	pinSingleConnection(t, db)
	tenantId := uuid.New()

	seedQuest(t, db, tenantId, 1, 100, StateStarted)
	seedQuest(t, db, tenantId, 1, 101, StateStarted)
	seedQuest(t, db, tenantId, 1, 102, StateCompleted)

	srv := httptest.NewServer(setupQuestRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/1/quests/started?page[number]=1&page[size]=250", srv.URL)
	req := requestQuestsWithTenant(http.MethodGet, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 2, "must exclude the completed quest")
	require.NotNil(t, doc.Meta)
	assert.EqualValues(t, 2, doc.Meta["total"])
}

// TestGetQuestProgressPaginates drives GET
// /characters/{characterId}/quests/{questId}/progress, verifying the
// paginated envelope AND that pagination is deterministic even though the
// underlying q.Progress() sub-list is preloaded with no explicit ORDER BY
// (task-117 determinism requirement -- seeded out of Id order so an
// unsorted Slice would fail this assertion).
func TestGetQuestProgressPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, progress.Migration)
	pinSingleConnection(t, db)
	tenantId := uuid.New()

	q := seedQuest(t, db, tenantId, 1, 100, StateStarted)

	// Seed progress rows with Ids assigned out of natural/insertion order by
	// creating them in a scrambled sequence (3, 1, 2) via explicit Id
	// values, proving the handler sorts rather than relying on insertion or
	// preload order.
	for _, id := range []uint32{3, 1, 2} {
		require.NoError(t, db.Create(&progress.Entity{
			ID:            id,
			TenantId:      tenantId,
			QuestStatusId: q.ID,
			InfoNumber:    id,
			Progress:      fmt.Sprintf("progress-%d", id),
		}).Error)
	}

	srv := httptest.NewServer(setupQuestRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/1/quests/100/progress?page[number]=1&page[size]=2", srv.URL)
	req := requestQuestsWithTenant(http.MethodGet, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var raw struct {
		Data []struct {
			Attributes struct {
				InfoNumber uint32 `json:"infoNumber"`
			} `json:"attributes"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))

	require.Len(t, raw.Data, 2)
	assert.Equal(t, 3, raw.Meta.Total)
	// Sorted by progress Id ascending -> InfoNumber 1 then 2 on page 1
	// (InfoNumber was set equal to Id above), NOT the scrambled seed order.
	assert.Equal(t, uint32(1), raw.Data[0].Attributes.InfoNumber)
	assert.Equal(t, uint32(2), raw.Data[1].Attributes.InfoNumber)
}
