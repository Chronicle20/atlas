package asset

import (
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newAssetTenantDB seeds two asset rows in two tenants sharing the same
// StorageId. Auto-increment IDs are globally unique under sqlite so we use 1
// and 2. Tenant scoping is asserted via GetByStorageId scoping each tenant
// to only its own row.
func newAssetTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	storageId := uuid.New()
	exp := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.Create(&Entity{
		Id: 1, TenantId: tidA, StorageId: storageId,
		InventoryType: 4, Slot: 0, TemplateId: 2000000,
		Expiration: exp, Quantity: 10,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Id: 2, TenantId: tidB, StorageId: storageId,
		InventoryType: 4, Slot: 0, TemplateId: 2000000,
		Expiration: exp, Quantity: 99,
	}).Error)
	return db, tidA, tidB, storageId
}

func TestAssetProvider_GetByStorageId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, storageId := newAssetTenantDB(t)

	gotA, err := GetByStorageId(db.WithContext(databasetest.TenantContext(tidA)))(storageId)
	require.NoError(t, err)
	require.Len(t, gotA, 1, "tenant A should only see its own asset")
	assert.Equal(t, uint32(1), gotA[0].Id())
	assert.Equal(t, uint32(10), gotA[0].Quantity())

	gotB, err := GetByStorageId(db.WithContext(databasetest.TenantContext(tidB)))(storageId)
	require.NoError(t, err)
	require.Len(t, gotB, 1, "tenant B should only see its own asset")
	assert.Equal(t, uint32(2), gotB[0].Id())
	assert.Equal(t, uint32(99), gotB[0].Quantity())
}

func TestAssetAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _, _ := newAssetTenantDB(t)

	err := db.WithContext(databasetest.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", 1).
		Update("quantity", uint32(7777)).Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, uint32(7777), r.Quantity)
		} else {
			assert.NotEqual(t, uint32(7777), r.Quantity, "tenant B must be untouched")
		}
	}
}
