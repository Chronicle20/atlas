package account_test

import (
	"atlas-account/account"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupAccountRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	account.InitRegistry(client)
}

func setupAccountRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := account.InitResource(&testServerInformation{})(db)
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

func seedAccount(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, name string) {
	t.Helper()
	require.NoError(t, db.Create(&account.Entity{ID: id, TenantId: tenantId, Name: name, Password: "pw"}).Error)
}

// TestGetAccountsPaginates drives GET /accounts through the real resource
// router (InitResource) against an in-memory tenant-scoped DB, verifying the
// JSON:API paginated envelope: page-size slicing, meta.total/meta.page.last,
// links.next/links.prev, and 400 on invalid paging params.
func TestGetAccountsPaginates(t *testing.T) {
	setupAccountRegistry(t)

	db := databasetest.NewInMemoryTenantDB(t, account.Migration)
	tenantId := uuid.New()
	seedAccount(t, db, tenantId, 1, "hero1")
	seedAccount(t, db, tenantId, 2, "hero2")
	seedAccount(t, db, tenantId, 3, "hero3")

	srv := httptest.NewServer(setupAccountRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/accounts/?page[number]=99&page[size]=2", srv.URL)
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
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
		assert.NotContains(t, doc.Links, "next")
	})
}

// TestCharacterSlotsResource drives GET/POST
// accounts/{accountId}/worlds/{worldId}/character-slots through the real
// resource router (task-246 bug-b-type-must-add-a-slot.md F1).
func TestCharacterSlotsResource(t *testing.T) {
	setupAccountRegistry(t)

	db := databasetest.NewInMemoryTenantDB(t, account.Migration)
	tenantId := uuid.New()
	seedAccount(t, db, tenantId, 1, "hero1")

	srv := httptest.NewServer(setupAccountRouter(db))
	defer srv.Close()

	getSlots := func(t *testing.T) (int, int16) {
		t.Helper()
		url := fmt.Sprintf("%s/accounts/1/worlds/0/character-slots", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, 0
		}
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		var attrs struct {
			Slots int16 `json:"slots"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
		return resp.StatusCode, attrs.Slots
	}

	postIncrement := func(t *testing.T) (int, int16) {
		t.Helper()
		url := fmt.Sprintf("%s/accounts/1/worlds/0/character-slots", srv.URL)
		req := requestWithTenant(http.MethodPost, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, 0
		}
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		var attrs struct {
			Slots int16 `json:"slots"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
		return resp.StatusCode, attrs.Slots
	}

	t.Run("DefaultsToFourWithoutARow", func(t *testing.T) {
		status, slots := getSlots(t)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 4, slots)
	})

	t.Run("IncrementPersists", func(t *testing.T) {
		status, slots := postIncrement(t)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 5, slots)

		status, slots = getSlots(t)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 5, slots)
	})

	t.Run("RejectsIncrementAtCap", func(t *testing.T) {
		for i := 0; i < 7; i++ {
			status, _ := postIncrement(t)
			require.Equal(t, http.StatusOK, status)
		}
		status, slots := getSlots(t)
		require.Equal(t, http.StatusOK, status)
		require.EqualValues(t, 12, slots)

		status, _ = postIncrement(t)
		assert.Equal(t, http.StatusConflict, status)

		status, slots = getSlots(t)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 12, slots)
	})
}
