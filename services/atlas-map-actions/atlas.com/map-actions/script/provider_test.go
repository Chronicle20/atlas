package script

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newScriptsDB seeds two script rows in two tenants that overlap on ScriptName
// and ScriptType. The primary key is a uuid so the two rows use distinct ids,
// and Data differs so a leak across the tenant boundary is observable through
// both the read and write paths. Note: the entity field name is TenantID
// (capital D) — GORM normalizes to the tenant_id column the callback expects.
func newScriptsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, MigrateTable)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: idA, TenantID: tidA, ScriptName: "doorway", ScriptType: "map_entry",
		Data: `{"scriptName":"doorway","rules":[]}`, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: idB, TenantID: tidB, ScriptName: "doorway", ScriptType: "map_entry",
		Data: `{"scriptName":"doorway","rules":[{"id":"tenantB"}]}`, CreatedAt: now, UpdatedAt: now,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestScriptProvider_ByNameAndType_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newScriptsDB(t)

	gotA, err := getByScriptNameAndTypeProvider("doorway")("map_entry")(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantID)
	assert.Equal(t, idA, gotA.ID)

	gotB, err := getByScriptNameAndTypeProvider("doorway")("map_entry")(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantID)
	assert.Equal(t, idB, gotB.ID)
}

func TestScriptAdministrator_DeleteMapScript_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, _, idB := newScriptsDB(t)

	// Tenant A asks to delete idB (which belongs to tenant B). With tenant
	// scoping nothing should be deleted — tenant B's row survives. deleteMapScript
	// is a soft delete, so we check that the row is still present and undeleted.
	require.NoError(t, deleteMapScript(db.WithContext(database.TenantContext(tidA)))(idB))

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		assert.False(t, r.DeletedAt.Valid, "no row should have been soft-deleted across tenants: id=%s", r.ID)
		if r.TenantID == tidB {
			assert.Equal(t, idB, r.ID)
		}
	}
}
