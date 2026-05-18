package history

import (
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newHistoryDB seeds two login-history rows in two tenants that overlap on
// AccountId and IPAddress so a leak across the tenant boundary is observable.
// The primary key is autoincrement and sqlite enforces global uniqueness, so
// the two rows use different ids (1 and 2).
func newHistoryDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidA, AccountId: 42, AccountName: "hero", IPAddress: "10.0.0.1", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&Entity{ID: 2, TenantId: tidB, AccountId: 42, AccountName: "hero", IPAddress: "10.0.0.1", CreatedAt: now}).Error)
	return db, tidA, tidB
}

func TestHistoryProvider_EntitiesByAccountId_FiltersByTenant(t *testing.T) {
	db, tidA, _ := newHistoryDB(t)

	rows, err := entitiesByAccountId(42)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rows, 1, "account 42 has rows in both tenants — only tenant A's row should return")
	assert.Equal(t, tidA, rows[0].TenantId)
}

func TestHistoryAdministrator_DeleteOlderThan_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newHistoryDB(t)

	// Use a cutoff in the future so both rows would be older — without tenant
	// scoping the call would delete both. The tenant callback should restrict
	// the delete to tenant A's row only.
	cutoff := time.Now().Add(1 * time.Hour)
	err := deleteOlderThan(db.WithContext(databasetest.TenantContext(tidA)))(cutoff)
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("id").Find(&rows).Error)
	require.Len(t, rows, 1, "tenant B's row must survive — only tenant A's row should have been deleted")
	assert.Equal(t, tidB, rows[0].TenantId)
}
