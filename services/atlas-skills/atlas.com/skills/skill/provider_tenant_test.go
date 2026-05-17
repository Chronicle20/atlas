package skill

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newSkillTenantDB seeds two skill rows in two tenants sharing the same
// CharacterId. The PK is the skill Id itself, so the two rows must use
// different Ids; tenant scoping is asserted via getByCharacterId scoping
// each tenant to only its own row.
func newSkillTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	exp := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.Create(&Entity{
		Id: 1001, TenantId: tidA, CharacterId: 500,
		Level: 1, MasterLevel: 20, Expiration: exp,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Id: 1002, TenantId: tidB, CharacterId: 500,
		Level: 1, MasterLevel: 20, Expiration: exp,
	}).Error)
	return db, tidA, tidB
}

func TestSkillProvider_GetByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newSkillTenantDB(t)

	gotA, err := getByCharacterId(500)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, gotA, 1, "tenant A should only see its own skill row")
	assert.Equal(t, tidA, gotA[0].TenantId)
	assert.Equal(t, uint32(1001), gotA[0].Id)

	gotB, err := getByCharacterId(500)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, gotB, 1, "tenant B should only see its own skill row")
	assert.Equal(t, tidB, gotB[0].TenantId)
	assert.Equal(t, uint32(1002), gotB[0].Id)
}

func TestSkillAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newSkillTenantDB(t)

	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("character_id = ?", 500).
		Update("level", byte(99)).Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, byte(99), r.Level)
		} else {
			assert.NotEqual(t, byte(99), r.Level, "tenant B must be untouched")
		}
	}
}
