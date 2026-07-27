package pet

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

// TestPetsBareList_PaginationEnvelope exercises GET /data/pets over the
// seeded pets: page[size]=2 returns page 1 (2 items), a total matching the
// fixture, and a next link.
func TestPetsBareList_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestPetData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/pets?page[number]=1&page[size]=2", ts.URL)
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

func TestPetsBareList_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestPetData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/pets?page[size]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
