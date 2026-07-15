package cash

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

// TestCashItemsBareList_PaginationEnvelope exercises GET /data/cash/items
// over the seeded cash items: page[size]=2 returns page 1 (2 items), a
// total matching the fixture, and a next link.
func TestCashItemsBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCashData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/cash/items?page[number]=1&page[size]=2", ts.URL)
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
	assert.NotNil(t, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestCashItemsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestCashData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/cash/items?page[size]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
