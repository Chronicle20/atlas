package storage

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newStorageTenantDB seeds two storage rows in two tenants sharing the same
// (WorldId, AccountId). The unique index includes tenant_id, so this is
// allowed. UUID PK is set explicitly to skip the BeforeCreate generator path.
func newStorageTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&Entity{
		Id: idA, TenantId: tidA, WorldId: 0, AccountId: 1000,
		Capacity: 4, Mesos: 100,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Id: idB, TenantId: tidB, WorldId: 0, AccountId: 1000,
		Capacity: 4, Mesos: 200,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func testLogger() logrus.FieldLogger {
	l, _ := logtest.NewNullLogger()
	return l
}

func TestStorageProvider_GetByWorldAndAccountId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newStorageTenantDB(t)

	gotA, err := GetByWorldAndAccountId(testLogger(),
		db.WithContext(database.TenantContext(tidA)))(0, 1000)
	require.NoError(t, err)
	assert.Equal(t, idA, gotA.Id())

	gotB, err := GetByWorldAndAccountId(testLogger(),
		db.WithContext(database.TenantContext(tidB)))(0, 1000)
	require.NoError(t, err)
	assert.Equal(t, idB, gotB.Id())
}

func TestStorageAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _, _, _ := newStorageTenantDB(t)

	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("account_id = ?", 1000).
		Update("mesos", uint32(9999)).Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, uint32(9999), r.Mesos)
		} else {
			assert.NotEqual(t, uint32(9999), r.Mesos, "tenant B must be untouched")
		}
	}
}
