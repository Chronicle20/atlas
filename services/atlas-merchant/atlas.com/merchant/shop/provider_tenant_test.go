package shop

import (
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newShopTenantDB seeds two shop rows in two tenants that overlap on
// CharacterId. The primary key is a UUID, so we generate distinct ids per row.
func newShopTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		Model:        gorm.Model{CreatedAt: now, UpdatedAt: now},
		Id:           idA,
		TenantId:     tidA,
		CharacterId:  1001,
		ShopType:     byte(CharacterShop),
		State:        byte(Draft),
		Title:        "tenantA",
		MapId:        910000001,
		InstanceId:   uuid.Nil,
		PermitItemId: 5030000,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Model:        gorm.Model{CreatedAt: now, UpdatedAt: now},
		Id:           idB,
		TenantId:     tidB,
		CharacterId:  1001,
		ShopType:     byte(CharacterShop),
		State:        byte(Draft),
		Title:        "tenantB",
		MapId:        910000001,
		InstanceId:   uuid.Nil,
		PermitItemId: 5030000,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestShopProvider_GetByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newShopTenantDB(t)

	gotA, err := getByCharacterId(1001)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, tidA, gotA[0].TenantId)
	assert.Equal(t, idA, gotA[0].Id)

	gotB, err := getByCharacterId(1001)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, tidB, gotB[0].TenantId)
	assert.Equal(t, idB, gotB[0].Id)
}

func TestShopAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, idA, _ := newShopTenantDB(t)

	// Load tenant A's row via tenant context, mutate Title, and Save().
	// The tenant callback must keep tenant B's row untouched.
	var entityA Entity
	require.NoError(t, db.WithContext(databasetest.TenantContext(tidA)).Where("id = ?", idA).First(&entityA).Error)
	entityA.Title = "tenantA-only"
	_, err := update(&entityA)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantId {
		case tidA:
			assert.Equal(t, "tenantA-only", r.Title, "tenant A's shop should be updated")
		case tidB:
			assert.Equal(t, "tenantB", r.Title, "tenant B must be untouched")
		}
	}
}
