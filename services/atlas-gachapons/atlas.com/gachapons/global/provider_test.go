package global

import (
	"testing"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newGlobalDB seeds two global-item rows in two tenants that overlap on Tier
// ("common") and ItemId. The autoincrement primary key is globally unique under
// sqlite, so the two rows use ids 1 and 2.
func newGlobalDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&entity{ID: 1, TenantId: tidA, ItemId: 2000000, Quantity: 1, Tier: "common"}).Error)
	require.NoError(t, db.Create(&entity{ID: 2, TenantId: tidB, ItemId: 2000000, Quantity: 1, Tier: "common"}).Error)
	return db, tidA, tidB
}

func TestGlobalProvider_GetByTier_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newGlobalDB(t)

	rowsA, err := getByTier("common")(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rowsA, 1, "tenant A must only see its own common-tier row")
	assert.Equal(t, tidA, rowsA[0].TenantId)
	assert.Equal(t, uint32(1), rowsA[0].ID)

	rowsB, err := getByTier("common")(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rowsB, 1)
	assert.Equal(t, tidB, rowsB[0].TenantId)
	assert.Equal(t, uint32(2), rowsB[0].ID)
}

func TestGlobalAdministrator_DeleteItem_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newGlobalDB(t)

	// Tenant A asks to delete id=2 (which belongs to tenant B). With tenant
	// scoping nothing should be deleted; tenant B's row survives.
	require.NoError(t, DeleteItem(db.WithContext(databasetest.TenantContext(tidA)), 2))

	var rows []entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2, "tenant A must not be able to delete tenant B's row")

	// Sanity check: tenant A deleting its own row works.
	require.NoError(t, DeleteItem(db.WithContext(databasetest.TenantContext(tidA)), 1))
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, tidB, rows[0].TenantId)
}
