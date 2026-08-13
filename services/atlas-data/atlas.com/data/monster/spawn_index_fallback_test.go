package monster

import (
	"atlas-data/canonical"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMonsterMaps_CanonicalFallback is the monster-side half of issue #1213:
// monster_spawn_index rows are written under the version-scoped canonical tenant
// during map ingest, so a real tenant scoped strictly to its own tenant_id read
// an always-empty partition.
func TestMonsterMaps_CanonicalFallback(t *testing.T) {
	db := setupResourceTestDB(t)
	require.NoError(t, db.AutoMigrate(&testMonsterSpawnIndexEntity{}))
	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	canonicalId := canonical.TenantId("GMS", 83, 1)
	require.NoError(t, db.Create(&[]SpawnIndexEntity{
		{TenantId: canonicalId, MonsterId: 200100, MapId: 100000000, Name: "Henesys", StreetName: "Victoria Road", SpawnCount: 5, UpdatedAt: time.Now()},
		{TenantId: canonicalId, MonsterId: 200100, MapId: 101000000, Name: "Ellinia", StreetName: "Victoria Road", SpawnCount: 2, UpdatedAt: time.Now()},
	}).Error)

	t.Run("TenantWithoutOwnRowsReadsCanonical", func(t *testing.T) {
		resp, err := http.DefaultClient.Do(createRequestWithTenant("GET",
			fmt.Sprintf("%s/data/monsters/200100/maps", ts.URL), uuid.New()))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		// MonsterSpawnMapRestModel carries mapId as the JSON:API resource id.
		var doc struct {
			Data []struct {
				Id         string `json:"id"`
				Attributes struct {
					Name       string `json:"name"`
					SpawnCount uint32 `json:"spawnCount"`
				} `json:"attributes"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.Len(t, doc.Data, 2)
		assert.Equal(t, "100000000", doc.Data[0].Id)
		assert.EqualValues(t, 5, doc.Data[0].Attributes.SpawnCount)
		assert.Equal(t, "101000000", doc.Data[1].Id)
	})

	t.Run("TenantWithOwnRowsDoesNotReadCanonical", func(t *testing.T) {
		ownTenantId := uuid.New()
		require.NoError(t, db.Create(&[]SpawnIndexEntity{
			{TenantId: ownTenantId, MonsterId: 200100, MapId: 200000000, Name: "Orbis", StreetName: "Orbis", SpawnCount: 9, UpdatedAt: time.Now()},
		}).Error)

		resp, err := http.DefaultClient.Do(createRequestWithTenant("GET",
			fmt.Sprintf("%s/data/monsters/200100/maps", ts.URL), ownTenantId))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc struct {
			Data []struct {
				Id         string `json:"id"`
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.Len(t, doc.Data, 1)
		assert.Equal(t, "200000000", doc.Data[0].Id)
		assert.Equal(t, "Orbis", doc.Data[0].Attributes.Name)
	})
}
