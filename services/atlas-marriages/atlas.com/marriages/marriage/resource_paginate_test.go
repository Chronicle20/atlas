package marriage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetMarriageHistoryPaginates exercises the paginate.ParseParams /
// server.MarshalPaginatedResponse envelope on GET
// /characters/{characterId}/marriage/history.
func TestGetMarriageHistoryPaginates(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	now := time.Now()

	// Seed 3 marriages for character 100, each against a different partner,
	// with created_at spread out so "created_at DESC" ordering is
	// unambiguous.
	for i := 0; i < 3; i++ {
		e := Entity{
			ID:           uint32(i + 1),
			CharacterId1: 100,
			CharacterId2: uint32(200 + i),
			Status:       StatusDivorced,
			ProposedAt:   now.Add(-time.Duration(72-i) * time.Hour),
			EngagedAt:    &now,
			MarriedAt:    &now,
			DivorcedAt:   &now,
			TenantId:     tenantId,
			CreatedAt:    now.Add(-time.Duration(72-i) * time.Hour),
			UpdatedAt:    now,
		}
		require.NoError(t, db.Create(&e).Error)
	}

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/100/marriage/history?page[number]=1&page[size]=2", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
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
		url := fmt.Sprintf("%s/characters/100/marriage/history?page[size]=0", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/100/marriage/history?limit=5", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/100/marriage/history?page[number]=99&page[size]=2", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
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

// TestGetProposalsPaginates exercises the same envelope on GET
// /characters/{characterId}/marriage/proposals, and proves the expiry
// filter (originally applied in Go, moved into the paged provider's SQL
// WHERE) still runs BEFORE pagination — an already-expired pending-status
// row must never occupy a page slot or inflate meta.total.
func TestGetProposalsPaginates(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	now := time.Now()

	// 3 live pending proposals for character 400 (as proposer), plus 1
	// EXPIRED pending-status proposal that must be excluded entirely.
	for i := 0; i < 3; i++ {
		e := ProposalEntity{
			ID:         uint32(i + 1),
			ProposerId: 400,
			TargetId:   uint32(500 + i),
			Status:     ProposalStatusPending,
			ProposedAt: now.Add(-time.Duration(i+1) * time.Hour),
			ExpiresAt:  now.Add(time.Duration(24-i) * time.Hour),
			TenantId:   tenantId,
			CreatedAt:  now.Add(-time.Duration(i+1) * time.Hour),
			UpdatedAt:  now.Add(-time.Duration(i+1) * time.Hour),
		}
		require.NoError(t, db.Create(&e).Error)
	}
	expired := ProposalEntity{
		ID:         99,
		ProposerId: 400,
		TargetId:   600,
		Status:     ProposalStatusPending,
		ProposedAt: now.Add(-48 * time.Hour),
		ExpiresAt:  now.Add(-1 * time.Hour), // already expired, still status=pending
		TenantId:   tenantId,
		CreatedAt:  now.Add(-48 * time.Hour),
		UpdatedAt:  now.Add(-48 * time.Hour),
	}
	require.NoError(t, db.Create(&expired).Error)

	router := setupTestRouter(db)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/400/marriage/proposals?page[number]=1&page[size]=2", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		// total must be 3 (the expired row excluded), not 4.
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/400/marriage/proposals?page[size]=0", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/400/marriage/proposals?limit=5", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/400/marriage/proposals?page[number]=99&page[size]=2", testServer.URL)
		req := createRequestWithTenant(http.MethodGet, url, nil, tenantId)
		resp, err := http.DefaultClient.Do(req)
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
