package reward

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/global"
	"atlas-reward-pools/item"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

func setupRewardRouter(db *gorm.DB) *mux.Router {
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

func seedMachineItem(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, gachaponId string, itemId uint32, tier string) {
	t.Helper()
	m, err := item.NewBuilder(tenantId, id).
		SetGachaponId(gachaponId).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier(tier).
		Build()
	require.NoError(t, err)
	require.NoError(t, item.CreateItem(db, m))
}

func seedGlobalItemRow(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, itemId uint32, tier string) {
	t.Helper()
	m, err := global.NewBuilder(tenantId, id).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier(tier).
		Build()
	require.NoError(t, err)
	require.NoError(t, global.CreateItem(db, m))
}

// TestGetPrizePoolPaginates drives GET /gachapons/{gachaponId}/prize-pool,
// the in-memory-merged (machine items + global items) arm, verifying the
// paginated envelope and deterministic ordering across pages (regression
// coverage for the stable-sort-before-slice recipe).
func TestGetPrizePoolPaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, gachapon.Migration, item.Migration, global.Migration)
	tenantId := uuid.New()

	g, err := gachapon.NewBuilder(tenantId, "henesys").
		SetName("Henesys Gachapon").
		SetNpcIds([]uint32{9100100}).
		SetCommonWeight(100).
		Build()
	require.NoError(t, err)
	require.NoError(t, gachapon.CreateGachapon(db, g))

	seedMachineItem(t, db, tenantId, 1, "henesys", 2000002, "common")
	seedMachineItem(t, db, tenantId, 2, "henesys", 2000000, "common")
	seedGlobalItemRow(t, db, tenantId, 1, 2000001, "common")
	// Noise: a different tier must not appear in the tier=common total.
	seedMachineItem(t, db, tenantId, 3, "henesys", 3000000, "rare")

	srv := httptest.NewServer(setupRewardRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwoOrderedByItemId", func(t *testing.T) {
		url := fmt.Sprintf("%s/gachapons/henesys/prize-pool?tier=common&page[number]=1&page[size]=2", srv.URL)
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
		assert.EqualValues(t, 3, doc.Meta["total"], "must include both machine and global common-tier items, exclude rare")

		var first, second map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.EqualValues(t, 2000000, first["itemId"], "stable sort by itemId must put the lowest itemId first")
		assert.EqualValues(t, 2000001, second["itemId"])
	})

	t.Run("SecondPageContinuesOrdering", func(t *testing.T) {
		url := fmt.Sprintf("%s/gachapons/henesys/prize-pool?tier=common&page[number]=2&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 1)
		var third map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &third))
		assert.EqualValues(t, 2000002, third["itemId"])
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/gachapons/henesys/prize-pool?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
