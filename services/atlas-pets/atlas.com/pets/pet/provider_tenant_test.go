package pet

import (
	"testing"
	"time"

	"atlas-pets/pet/exclude"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newPetTenantDB seeds two pet rows in two tenants. Auto-increment IDs are
// globally unique under sqlite so we use 1 and 2.
func newPetTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration, exclude.Migration)
	tidA, tidB := uuid.New(), uuid.New()
	slot := int8(-1)
	exp := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.Create(&Entity{
		Id: 1, TenantId: tidA, OwnerId: 100, CashId: 7000001,
		TemplateId: 5000017, Name: "petA", Expiration: exp, Slot: &slot,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Id: 2, TenantId: tidB, OwnerId: 100, CashId: 7000002,
		TemplateId: 5000017, Name: "petB", Expiration: exp, Slot: &slot,
	}).Error)
	return db, tidA, tidB
}

func TestPetProvider_GetById_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newPetTenantDB(t)

	gotA, err := getById(1)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, uint32(1), gotA.Id)

	// Tenant B asking for tenant A's row must miss.
	_, err = getById(1)(db.WithContext(databasetest.TenantContext(tidB)))()
	assert.Error(t, err)
}

func TestPetAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newPetTenantDB(t)

	err := db.WithContext(databasetest.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", 1).
		Update("name", "tenantA-only").Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, "tenantA-only", r.Name)
		} else {
			assert.NotEqual(t, "tenantA-only", r.Name, "tenant B must be untouched")
		}
	}
}
