package family

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string   { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupFamilyRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedFamilyMember(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id, characterId uint32, seniorId *uint32, juniorIds []uint32) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: id, TenantId: tenantId, CharacterId: characterId, SeniorId: seniorId,
		JuniorIds: juniorIds, Level: 10, World: 0,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
}

// TestGetFamilyTreePaginates drives GET /families/tree/{characterId} through
// the real resource router (InitResource) against an in-memory tenant-scoped
// DB. The tree for character 20 is: self(20) + senior(10) + juniors of 20
// (30, 40) + sibling (21, another junior of 10) = 5 members, deterministically
// ordered by CharacterId (10, 20, 21, 30, 40) regardless of DB fetch order.
func TestGetFamilyTreePaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	ten := uint32(10)
	twenty := uint32(20)
	seedFamilyMember(t, db, tenantId, 1, 10, nil, []uint32{20, 21})
	seedFamilyMember(t, db, tenantId, 2, 20, &ten, []uint32{30, 40})
	seedFamilyMember(t, db, tenantId, 3, 21, &ten, nil)
	seedFamilyMember(t, db, tenantId, 4, 30, &twenty, nil)
	seedFamilyMember(t, db, tenantId, 5, 40, &twenty, nil)

	srv := httptest.NewServer(setupFamilyRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		// stable-sorted by CharacterId: page 1 must be {10, 20}, not
		// whatever order the graph traversal (self, senior, juniors,
		// siblings) happened to build the slice in.
		var firstAttrs, secondAttrs struct {
			CharacterId uint32 `json:"characterId"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &firstAttrs))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &secondAttrs))
		assert.EqualValues(t, 10, firstAttrs.CharacterId)
		assert.EqualValues(t, 20, secondAttrs.CharacterId)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 5, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 3, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

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
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=3")
		assert.NotContains(t, doc.Links, "next")
	})
}
