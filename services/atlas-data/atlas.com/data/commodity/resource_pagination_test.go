package commodity

import (
	"atlas-data/document"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCommodityItemsBareList_PaginationEnvelope exercises
// GET /data/commodity/items over the 3 seeded commodities
// (setupTestCommodityData): page[size]=2 returns page 1 (2 items), a total
// of 3, and a next link.
func TestCommodityItemsBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCommodityData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/commodity/items?page[number]=1&page[size]=2", ts.URL)
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

func TestCommodityItemsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCommodityData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/commodity/items?page[size]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestCommodityByItem_PaginationEnvelope exercises the by-item
// derived-filter arm: seeds a 4th commodity sharing itemId 5000000 with the
// first fixture row so the filtered set has 2 members.
func TestCommodityByItem_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCommodityData(t, db, tenantId)

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	storage := document.NewStorage(l, db, GetModelRegistry(), "COMMODITY")
	_, err = storage.Add(ctx)(RestModel{
		Id: 4, ItemId: 5000000, Count: 10, Price: 3900, Period: 30, Priority: 100, OnSale: true,
	})()
	require.NoError(t, err)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/commodity/by-item/5000000?page[number]=1&page[size]=1", ts.URL)
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

func TestCommodityByItem_RejectsBadPageNumber(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCommodityData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/commodity/by-item/5000000?page[number]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
