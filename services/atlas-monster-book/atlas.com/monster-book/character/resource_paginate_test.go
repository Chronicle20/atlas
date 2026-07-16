package character

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atlas-monster-book/card"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type paginateTestServerInformation struct{}

func (t *paginateTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *paginateTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &paginateTestServerInformation{}

func setupCardRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&paginateTestServerInformation{})(db)
	ri(r, l)
	return r
}

func requestCardsWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// cardEntity mirrors card's unexported entity struct (same package layout
// as monster_book_cards) so this test can seed rows directly without
// depending on card's own unexported literal type.
type cardEntity struct {
	TenantId        uuid.UUID `gorm:"primaryKey;autoIncrement:false;not null"`
	CharacterId     uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	CardId          uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	Level           uint8     `gorm:"not null"`
	IsSpecial       bool      `gorm:"not null;default:false;index"`
	FirstAcquiredAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (cardEntity) TableName() string { return "monster_book_cards" }

func seedCard(t *testing.T, db *gorm.DB, tenantId uuid.UUID, characterId uint32, cardId uint32, isSpecial bool) {
	t.Helper()
	require.NoError(t, db.Create(&cardEntity{
		TenantId:    tenantId,
		CharacterId: characterId,
		CardId:      cardId,
		Level:       1,
		IsSpecial:   isSpecial,
	}).Error)
}

// TestGetCardsPaginates drives GET
// /characters/{characterId}/monster-book/cards through the real resource
// router, verifying the JSON:API paginated envelope, 400 on invalid paging
// params, empty-page handling past the last page, and that another
// character's cards are excluded from the total. Also proves determinism:
// cards are seeded with out-of-order CardIds (30,10,20) and the handler's
// CardId sort (over the composite-PK entity, which has no ORDER BY of its
// own) must still return them in ascending order across pages.
func TestGetCardsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, card.Migration)
	tenantId := uuid.New()

	// cardIds must classify as ClassificationConsumableMonsterCard (item.Id
	// / 10000 == 238, see IsCardId/Make/Build) or the transform step 500s.
	// >= 2388000 is the special-card range (IsSpecialCard).
	seedCard(t, db, tenantId, 1, 2380030, false)
	seedCard(t, db, tenantId, 1, 2380010, false)
	seedCard(t, db, tenantId, 1, 2388005, true)
	seedCard(t, db, tenantId, 2, 2380999, false)

	srv := httptest.NewServer(setupCardRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwoSortedByCardId", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/monster-book/cards?page[number]=1&page[size]=2", srv.URL)
		req := requestCardsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var raw struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
			Meta struct {
				Total int `json:"total"`
				Page  struct {
					Last int `json:"last"`
				} `json:"page"`
			} `json:"meta"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))

		require.Len(t, raw.Data, 2)
		// Seeded out of order (2380030, 2380010, 2388005); sorted ascending
		// by CardId -> 2380010, 2380030 on page 1.
		assert.Equal(t, "2380010", raw.Data[0].Id)
		assert.Equal(t, "2380030", raw.Data[1].Id)
		assert.Equal(t, 3, raw.Meta.Total, "must exclude character 2's card")
		assert.Equal(t, 2, raw.Meta.Page.Last)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/monster-book/cards?page[size]=0", srv.URL)
		req := requestCardsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/monster-book/cards?limit=5", srv.URL)
		req := requestCardsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/monster-book/cards?page[number]=99&page[size]=2", srv.URL)
		req := requestCardsWithTenant(http.MethodGet, url, tenantId)

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

	t.Run("FilterIsSpecialAppliedBeforePagination", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/1/monster-book/cards?filter[isSpecial]=true&page[number]=1&page[size]=250", srv.URL)
		req := requestCardsWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 1, "only cardId 2388005 is special")
		assert.Equal(t, "2388005", doc.Data.DataArray[0].ID)
	})
}
