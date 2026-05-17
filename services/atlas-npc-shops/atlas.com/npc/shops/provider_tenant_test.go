package shops

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newShopsTenantDB seeds two shop rows in two tenants that share the same
// NpcId. UUID primary keys are unique per row.
func newShopsTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		Model:     gorm.Model{CreatedAt: now, UpdatedAt: now},
		Id:        idA,
		TenantId:  tidA,
		NpcId:     9201000,
		Recharger: false,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Model:     gorm.Model{CreatedAt: now, UpdatedAt: now},
		Id:        idB,
		TenantId:  tidB,
		NpcId:     9201000,
		Recharger: false,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestShopsProvider_GetByNpcId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newShopsTenantDB(t)

	gotA, err := getByNpcId(9201000)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, idA, gotA.Id)

	gotB, err := getByNpcId(9201000)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantId)
	assert.Equal(t, idB, gotB.Id)
}

func TestShopsAdministrator_UpdateShop_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, _, _ := newShopsTenantDB(t)

	// updateShop loads-by-npcId then Save()s — the tenant callback must keep
	// tenant B's matching-npcId row untouched.
	_, err := updateShop(tidA, 9201000, true)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantId {
		case tidA:
			assert.True(t, r.Recharger, "tenant A's shop should be flipped to recharger")
		case tidB:
			assert.False(t, r.Recharger, "tenant B must be untouched")
		}
	}
}
