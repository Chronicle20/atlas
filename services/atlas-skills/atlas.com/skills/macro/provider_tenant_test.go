package macro

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newMacroTenantDB seeds two macro rows in two tenants sharing the same
// CharacterId. The PK is (character_id, id) without tenant_id, so the two
// rows must use different Ids; tenant scoping is asserted by ensuring each
// tenant only sees its own row.
func newMacroTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&Entity{
		TenantId: tidA, CharacterId: 500, Id: 1,
		Name: "macroA", Shout: false,
		SkillId1: 1000, SkillId2: 1001, SkillId3: 1002,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		TenantId: tidB, CharacterId: 500, Id: 2,
		Name: "macroB", Shout: false,
		SkillId1: 2000, SkillId2: 2001, SkillId3: 2002,
	}).Error)
	return db, tidA, tidB
}

func TestMacroProvider_GetByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newMacroTenantDB(t)

	gotA, err := getByCharacterId(500)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, gotA, 1, "tenant A should only see its own macro")
	assert.Equal(t, tidA, gotA[0].TenantId)
	assert.Equal(t, "macroA", gotA[0].Name)

	gotB, err := getByCharacterId(500)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, gotB, 1, "tenant B should only see its own macro")
	assert.Equal(t, tidB, gotB[0].TenantId)
	assert.Equal(t, "macroB", gotB[0].Name)
}

func TestMacroAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newMacroTenantDB(t)

	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("character_id = ?", 500).
		Update("name", "tenantA-only").Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, "tenantA-only", r.Name)
		} else {
			assert.NotEqual(t, "tenantA-only", r.Name, "tenant B must be untouched")
		}
	}
}
