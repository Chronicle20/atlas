package consumable

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

// TestConsumablesBareList_PaginationEnvelope exercises GET /data/consumables
// over the 4 seeded consumables (setupTestConsumableData): page[size]=2
// returns page 1 (2 items), a total of 4, and a next link.
func TestConsumablesBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestConsumableData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/consumables?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, nil, tenantId))
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
	assert.EqualValues(t, 4, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestConsumablesBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestConsumableData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/consumables?page[size]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, nil, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestConsumablesRechargeableFilter_PaginationEnvelope exercises the
// filter[rechargeable]=true derived-filter arm: 1 of the 4 fixture
// consumables is rechargeable, so total must be 1.
func TestConsumablesRechargeableFilter_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestConsumableData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/consumables?filter[rechargeable]=true&page[number]=1&page[size]=50", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, nil, tenantId))
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
	assert.EqualValues(t, 1, doc.Meta["total"])
	assert.Nil(t, doc.Links["next"])
}

func TestConsumablesRechargeableFilter_RejectsBadPageNumber(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestConsumableData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/consumables?filter[rechargeable]=true&page[number]=0", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, nil, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
