package ban

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newBansDB seeds two ban rows in two tenants that overlap on the Value field
// so a leak across the tenant boundary is observable. The primary key is
// autoincrement and sqlite enforces global uniqueness, so the two rows use
// different ids (1 and 2).
func newBansDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	expiresA := time.Now().Add(24 * time.Hour)
	expiresB := time.Now().Add(48 * time.Hour)
	require.NoError(t, db.Create(&Entity{ID: 1, TenantId: tidA, BanType: byte(BanTypeIP), Value: "10.0.0.1", ExpiresAt: expiresA}).Error)
	require.NoError(t, db.Create(&Entity{ID: 2, TenantId: tidB, BanType: byte(BanTypeIP), Value: "10.0.0.1", ExpiresAt: expiresB}).Error)
	return db, tidA, tidB
}

func TestBanProvider_EntityById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newBansDB(t)

	gotA, err := entityById(1)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)

	// Tenant B asking for id 1 (which belongs to tenant A) must not see it.
	_, err = entityById(1)(db.WithContext(database.TenantContext(tidB)))()
	require.Error(t, err, "tenant B must not see tenant A's row by id")
}

func TestBanAdministrator_UpdateExpiresAt_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newBansDB(t)

	newExpiry := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	// Tenant A updates ban id=1. The tenant callback must restrict the write to
	// tenant A's row only — even if a malicious actor tries to target id=2, the
	// matching tenant_id should prevent the write.
	err := updateExpiresAt(db.WithContext(database.TenantContext(tidA)))(1, newExpiry)
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.WithinDuration(t, newExpiry, r.ExpiresAt, time.Second, "tenant A's expiry should be updated")
		} else {
			assert.False(t, r.ExpiresAt.Equal(newExpiry), "tenant B must be untouched")
		}
	}
}
