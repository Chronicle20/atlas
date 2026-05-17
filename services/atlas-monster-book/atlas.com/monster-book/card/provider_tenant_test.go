package card

import (
	"testing"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newTenantDB seeds two card rows in two tenants that overlap on
// (CharacterId, CardId). The composite primary key includes TenantId so both
// can coexist.
func newTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&entity{
		TenantId: tidA, CharacterId: 1001, CardId: 2380000, Level: 1, IsSpecial: false,
	}).Error)
	require.NoError(t, db.Create(&entity{
		TenantId: tidB, CharacterId: 1001, CardId: 2380000, Level: 1, IsSpecial: false,
	}).Error)
	return db, tidA, tidB
}

func TestCardProvider_ByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newTenantDB(t)

	// byCharacterIdEntityProvider applies an explicit tenant_id filter via the
	// argument it receives — verify it returns only this tenant's rows.
	gotA, err := byCharacterIdEntityProvider(tidA, 1001)(db)()
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, tidA, gotA[0].TenantId)

	gotB, err := byCharacterIdEntityProvider(tidB, 1001)(db)()
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, tidB, gotB[0].TenantId)
}

func TestCardAdministrator_UpsertCard_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newTenantDB(t)

	// upsertCard must increment only the tenant-A row, leaving tenant B's at
	// level 1.
	eid := uuid.New()
	res, err := upsertCard(db.WithContext(databasetest.TenantContext(tidA)), tidA, 1001, 2380000, eid)
	require.NoError(t, err)
	require.False(t, res.Inserted)
	require.Equal(t, uint8(2), res.NewLevel)

	var rows []entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantId {
		case tidA:
			assert.Equal(t, uint8(2), r.Level, "tenant A's card should be leveled up")
		case tidB:
			assert.Equal(t, uint8(1), r.Level, "tenant B must be untouched")
		}
	}
}
