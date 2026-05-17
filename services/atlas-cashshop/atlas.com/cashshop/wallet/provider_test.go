package wallet

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newWalletsDB seeds two wallet rows in two tenants that overlap on AccountId
// so a leak across the tenant boundary is observable. The primary key is a
// uuid set explicitly (the BeforeCreate hook would also assign one), so there
// is no sqlite uniqueness conflict to worry about.
func newWalletsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&Entity{Id: uuid.New(), TenantId: tidA, AccountId: 42, Credit: 100, Points: 10, Prepaid: 1}).Error)
	require.NoError(t, db.Create(&Entity{Id: uuid.New(), TenantId: tidB, AccountId: 42, Credit: 500, Points: 50, Prepaid: 5}).Error)
	return db, tidA, tidB
}

func TestWalletProvider_ByAccountId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newWalletsDB(t)

	gotA, err := byAccountIdEntityProvider(42)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, uint32(100), gotA.Credit, "should be tenant A's wallet")

	gotB, err := byAccountIdEntityProvider(42)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantId)
	assert.Equal(t, uint32(500), gotB.Credit, "should be tenant B's wallet")
}

func TestWalletAdministrator_UpdateEntity_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newWalletsDB(t)

	// updateEntity reads then writes under the same context. Both halves must
	// scope to tenant A — read returns tenant A's row, save updates only that
	// row.
	_, err := updateEntity(db.WithContext(database.TenantContext(tidA)), 42, 999, 99, 9)
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, uint32(999), r.Credit, "tenant A's credit should be updated")
		} else {
			assert.Equal(t, uint32(500), r.Credit, "tenant B must be untouched")
		}
	}
}
