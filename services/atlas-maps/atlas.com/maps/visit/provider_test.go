package visit

import (
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newVisitsDB seeds two visit rows in two tenants that overlap fully on
// CharacterID and MapID. The composite unique index (tenant_id, character_id,
// map_id) admits the overlap so this two-tenant fixture exercises the read
// and delete-by-character-id write paths.
func newVisitsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, MigrateTable)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: idA, TenantId: tidA, CharacterID: 1001, MapID: 100000000, FirstVisitedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: idB, TenantId: tidB, CharacterID: 1001, MapID: 100000000, FirstVisitedAt: now,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestVisitProvider_ByCharacterIdAndMapId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newVisitsDB(t)

	gotA, err := getByCharacterIdAndMapIdProvider(1001)(_map.Id(100000000))(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, idA, gotA.ID)

	gotB, err := getByCharacterIdAndMapIdProvider(1001)(_map.Id(100000000))(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantId)
	assert.Equal(t, idB, gotB.ID)
}

func TestVisitAdministrator_DeleteByCharacterId_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, _, idB := newVisitsDB(t)

	// Tenant A deletes by character_id alone. Without tenant scoping the
	// underlying WHERE would purge both tenants' visits for character 1001.
	affected, err := deleteByCharacterId(db.WithContext(database.TenantContext(tidA)))(1001)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "only tenant A's visit should be deleted")

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 1, "tenant B's visit must survive tenant A's delete")
	assert.Equal(t, tidB, rows[0].TenantId)
	assert.Equal(t, idB, rows[0].ID)
}
