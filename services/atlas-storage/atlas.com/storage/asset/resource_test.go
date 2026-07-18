package asset

import (
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

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

// migrateStorages AutoMigrates the storages table (owned by the sibling
// storage package, but StorageEntity is mirrored here for the
// GetOrCreateStorageId cross-package query) so tests can pre-seed a storage
// row and avoid exercising the create-on-first-access path.
func migrateStorages(db *gorm.DB) error {
	return db.AutoMigrate(&StorageEntity{})
}

func setupAssetRouter(db *gorm.DB) *mux.Router {
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

func seedStorage(t *testing.T, db *gorm.DB, tenantId uuid.UUID, storageId uuid.UUID, worldId byte, accountId uint32) {
	t.Helper()
	require.NoError(t, db.Create(&StorageEntity{
		TenantId:  tenantId,
		Id:        storageId,
		WorldId:   worldId,
		AccountId: accountId,
		Capacity:  4,
	}).Error)
}

func seedStorageAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, storageId uuid.UUID, templateId uint32) {
	t.Helper()
	require.NoError(t, db.Create(&Entity{
		TenantId:      tenantId,
		StorageId:     storageId,
		InventoryType: 4,
		TemplateId:    templateId,
		Expiration:    time.Time{},
	}).Error)
}

// TestGetAssetsPaginates drives GET /storage/accounts/{accountId}/assets?worldId=N
// through the real resource router, verifying the JSON:API paginated
// envelope, 400 on invalid paging params, and empty-page handling past the
// last page. Also confirms a different storage's assets (different account)
// are excluded from the total.
func TestGetAssetsPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, migrateStorages)
	tenantId := uuid.New()
	storageId := uuid.New()
	otherStorageId := uuid.New()
	seedStorage(t, db, tenantId, storageId, 0, 1)
	seedStorage(t, db, tenantId, otherStorageId, 0, 2)
	seedStorageAsset(t, db, tenantId, storageId, 2000000)
	seedStorageAsset(t, db, tenantId, storageId, 2000001)
	seedStorageAsset(t, db, tenantId, storageId, 2000002)
	seedStorageAsset(t, db, tenantId, otherStorageId, 9999999)

	srv := httptest.NewServer(setupAssetRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/storage/accounts/1/assets?worldId=0&page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"], "must exclude the other account's storage assets")
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/storage/accounts/1/assets?worldId=0&page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/storage/accounts/1/assets?worldId=0&limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/storage/accounts/1/assets?worldId=0&page[number]=99&page[size]=2", srv.URL)
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

// TestGetAssetsDynamicSlotStableAcrossPages proves the dynamic-slot
// decoration (MakeWithDynamicSlot, applied over the FULL ordered asset list
// before paginate.Slice) does not restart at 0 on page 2 -- an
// OFFSET/LIMIT-pushed-into-SQL implementation would have assigned page 2's
// items slots 0..N again, colliding with page 1's slots.
func TestGetAssetsDynamicSlotStableAcrossPages(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, migrateStorages)
	tenantId := uuid.New()
	storageId := uuid.New()
	seedStorage(t, db, tenantId, storageId, 0, 1)
	for i := 0; i < 4; i++ {
		seedStorageAsset(t, db, tenantId, storageId, uint32(2000000+i))
	}

	srv := httptest.NewServer(setupAssetRouter(db))
	defer srv.Close()

	type assetAttrs struct {
		Slot int16 `json:"slot"`
	}

	fetchPage := func(number int) []assetAttrs {
		url := fmt.Sprintf("%s/storage/accounts/1/assets?worldId=0&page[number]=%d&page[size]=2", srv.URL, number)
		req := requestWithTenant(http.MethodGet, url, tenantId)
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var raw struct {
			Data []struct {
				Attributes assetAttrs `json:"attributes"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&raw))
		out := make([]assetAttrs, 0, len(raw.Data))
		for _, d := range raw.Data {
			out = append(out, d.Attributes)
		}
		return out
	}

	page1 := fetchPage(1)
	page2 := fetchPage(2)
	require.Len(t, page1, 2)
	require.Len(t, page2, 2)

	seen := map[int16]bool{}
	for _, a := range append(page1, page2...) {
		assert.False(t, seen[a.Slot], "slot %d must not repeat across pages", a.Slot)
		seen[a.Slot] = true
	}
	assert.ElementsMatch(t, []int16{0, 1, 2, 3}, []int16{page1[0].Slot, page1[1].Slot, page2[0].Slot, page2[1].Slot})
}
