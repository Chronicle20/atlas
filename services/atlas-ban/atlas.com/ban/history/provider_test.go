package history

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

	paged, err := entitiesByAccountId(42, model.Page{Number: 1, Size: 50})(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, paged.Items, 1, "account 42 has rows in both tenants — only tenant A's row should return")
	assert.Equal(t, tidA, paged.Items[0].TenantId)
	assert.Equal(t, 1, paged.Total)
}

// TestHistoryProvider_EntitiesByAccountId_OrderedByCreatedAtDesc pins the
// CRITICAL caller-order-preservation requirement: entitiesByAccountId's
// Order("created_at desc") must still be the effective ordering on page 1
// after PagedQuery appends its PK tie-break, per PagedQuery's documented
// "caller order preserved, PK order appended after" contract.
func TestHistoryProvider_EntitiesByAccountId_OrderedByCreatedAtDesc(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tid, AccountId: 42, AccountName: "hero", CreatedAt: now.Add(-2 * time.Hour)}).Error)
	require.NoError(t, db.Create(&Entity{ID: 2, TenantId: tid, AccountId: 42, AccountName: "hero", CreatedAt: now.Add(-1 * time.Hour)}).Error)
	require.NoError(t, db.Create(&Entity{ID: 3, TenantId: tid, AccountId: 42, AccountName: "hero", CreatedAt: now}).Error)

	paged, err := entitiesByAccountId(42, model.Page{Number: 1, Size: 2})(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	require.Len(t, paged.Items, 2)
	assert.Equal(t, uint64(3), paged.Items[0].ID, "page 1 should return the newest row first")
	assert.Equal(t, uint64(2), paged.Items[1].ID, "page 1 should return the second-newest row second")
	assert.Equal(t, 3, paged.Total)
}

// TestHistoryProvider_EntitiesByTenant_OrderedByCreatedAtDesc pins the same
// CRITICAL caller-order-preservation requirement for the bare /history/
// list's entitiesByTenant provider.
func TestHistoryProvider_EntitiesByTenant_OrderedByCreatedAtDesc(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tid, AccountId: 1, AccountName: "a", CreatedAt: now.Add(-2 * time.Hour)}).Error)
	require.NoError(t, db.Create(&Entity{ID: 2, TenantId: tid, AccountId: 2, AccountName: "b", CreatedAt: now.Add(-1 * time.Hour)}).Error)
	require.NoError(t, db.Create(&Entity{ID: 3, TenantId: tid, AccountId: 3, AccountName: "c", CreatedAt: now}).Error)

	paged, err := entitiesByTenant(model.Page{Number: 1, Size: 2})(db.WithContext(databasetest.TenantContext(tid)))()
	require.NoError(t, err)
	require.Len(t, paged.Items, 2)
	assert.Equal(t, uint64(3), paged.Items[0].ID, "page 1 should return the newest row first")
	assert.Equal(t, uint64(2), paged.Items[1].ID, "page 1 should return the second-newest row second")
	assert.Equal(t, 3, paged.Total)
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
