package compartment

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newCompartmentDB seeds two compartments in two tenants that overlap on
// CharacterId and InventoryType. The primary key is a uuid so the two rows use
// distinct ids; capacity differs so a leak across the tenant boundary is
// observable through either read or write paths.
func newCompartmentDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&Entity{
		Id: idA, TenantId: tidA, CharacterId: 1001, InventoryType: inventory.TypeValueEquip, Capacity: 24,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Id: idB, TenantId: tidB, CharacterId: 1001, InventoryType: inventory.TypeValueEquip, Capacity: 96,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestCompartmentProvider_GetByCharacterAndType_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newCompartmentDB(t)

	gotA, err := getByCharacterAndType(1001, inventory.TypeValueEquip)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, idA, gotA.Id)
	assert.Equal(t, uint32(24), gotA.Capacity)

	gotB, err := getByCharacterAndType(1001, inventory.TypeValueEquip)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantId)
	assert.Equal(t, idB, gotB.Id)
	assert.Equal(t, uint32(96), gotB.Capacity)
}

func TestCompartmentAdministrator_UpdateCapacity_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newCompartmentDB(t)

	// Tenant A updates capacity for character 1001's equip compartment. Without
	// tenant scoping the same character+type lookup would match tenant B's row
	// too (Save uses primary-key match by id afterwards, but the First() lookup
	// is what tenant scoping must protect here).
	updated, err := updateCapacity(db.WithContext(databasetest.TenantContext(tidA)), 1001, int8(inventory.TypeValueEquip), 48)
	require.NoError(t, err)
	assert.Equal(t, idA, updated.Id())
	assert.Equal(t, uint32(48), updated.Capacity())

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantId {
		case tidA:
			assert.Equal(t, idA, r.Id)
			assert.Equal(t, uint32(48), r.Capacity, "tenant A's capacity should be updated")
		case tidB:
			assert.Equal(t, idB, r.Id)
			assert.Equal(t, uint32(96), r.Capacity, "tenant B must be untouched")
		}
	}
}
