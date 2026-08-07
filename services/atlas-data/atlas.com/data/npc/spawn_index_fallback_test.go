package npc

import (
	"atlas-data/canonical"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNpcMaps_CanonicalFallback covers issue #1213: MAP documents (and the
// npc_spawn_index rows derived from them) are ingested under the version-scoped
// canonical tenant, so a real tenant scoped strictly to its own tenant_id read
// an always-empty partition and the UI's "Spawn Locations" card rendered its
// empty state on every tenant.
//
// The fallback mirrors document.Storage and searchindex.ResolveTenantId: use the
// tenant's own rows when it has any, otherwise the canonical partition.
func TestNpcMaps_CanonicalFallback(t *testing.T) {
	db := setupResourceTestDB(t)
	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	canonicalId := canonical.TenantId("GMS", 83, 1)
	require.NoError(t, db.Create(&[]testSpawnIndexEntity{
		{TenantId: canonicalId, NpcId: 9020000, MapId: 100000000, Name: "Henesys", StreetName: "Victoria Road", SpawnCount: 2},
		{TenantId: canonicalId, NpcId: 9020000, MapId: 101000000, Name: "Ellinia", StreetName: "Victoria Road", SpawnCount: 1},
	}).Error)

	t.Run("TenantWithoutOwnRowsReadsCanonical", func(t *testing.T) {
		resp, err := http.DefaultClient.Do(createRequestWithTenant("GET",
			fmt.Sprintf("%s/data/npcs/9020000/maps", ts.URL), uuid.New()))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc struct {
			Data []struct {
				Attributes struct {
					MapId      uint32 `json:"mapId"`
					Name       string `json:"name"`
					SpawnCount uint32 `json:"spawnCount"`
				} `json:"attributes"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.Len(t, doc.Data, 2)
		assert.EqualValues(t, 100000000, doc.Data[0].Attributes.MapId)
		assert.Equal(t, "Henesys", doc.Data[0].Attributes.Name)
		assert.EqualValues(t, 2, doc.Data[0].Attributes.SpawnCount)
		assert.EqualValues(t, 101000000, doc.Data[1].Attributes.MapId)
	})

	// A tenant that has ingested its own maps must keep reading its own rows —
	// the fallback must not merge the two partitions.
	t.Run("TenantWithOwnRowsDoesNotReadCanonical", func(t *testing.T) {
		ownTenantId := uuid.New()
		require.NoError(t, db.Create(&[]testSpawnIndexEntity{
			{TenantId: ownTenantId, NpcId: 9020000, MapId: 200000000, Name: "Orbis", StreetName: "Orbis", SpawnCount: 7},
		}).Error)

		resp, err := http.DefaultClient.Do(createRequestWithTenant("GET",
			fmt.Sprintf("%s/data/npcs/9020000/maps", ts.URL), ownTenantId))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc struct {
			Data []struct {
				Attributes struct {
					MapId uint32 `json:"mapId"`
					Name  string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.Len(t, doc.Data, 1)
		assert.EqualValues(t, 200000000, doc.Data[0].Attributes.MapId)
		assert.Equal(t, "Orbis", doc.Data[0].Attributes.Name)
	})
}
