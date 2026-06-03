package account

import (
	"testing"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newAccountsDB seeds two accounts in two tenants that overlap on the Name field
// so any leak across the tenant boundary is observable. The primary key is
// autoincrement and sqlite enforces global uniqueness, so the two rows use
// different ids (1 and 2).
func newAccountsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidA, Name: "hero", Password: "pwA", Gender: 0}).Error)
	require.NoError(t, db.Create(&Entity{ID: 2, TenantId: tidB, Name: "hero", Password: "pwB", Gender: 0}).Error)
	return db, tidA, tidB
}

func TestAccountProvider_EntityById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newAccountsDB(t)

	gotA, err := entityById(1)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, "pwA", gotA.Password)

	// Tenant B asking for id 1 (which belongs to tenant A) must not see it.
	_, err = entityById(1)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.Error(t, err, "tenant B must not see tenant A's row by id")
}

func TestAccountAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newAccountsDB(t)

	// Tenant A updates its own row by id. The update path uses the same gorm
	// context, so the tenant callback must scope the write to tenant A only.
	err := update(db.WithContext(databasetest.TenantContext(tidA)))(updatePic("tenantA-only"))(1)
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, "tenantA-only", r.PIC, "tenant A's pic should be set")
		} else {
			assert.NotEqual(t, "tenantA-only", r.PIC, "tenant B must be untouched")
		}
	}
}
