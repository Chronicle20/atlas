package list

import (
	"testing"

	"atlas-buddies/buddy"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newListsDB seeds two buddy lists in two tenants that overlap on CharacterId
// so a leak across the tenant boundary is observable. We set the list Id
// explicitly so the entity's `default:uuid_generate_v4()` (PostgreSQL-only)
// never fires on sqlite.
func newListsDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	// list.Migration uses PostgreSQL-specific uuid_generate_v4() and cannot run
	// on sqlite; create the lists table directly so the provider has a target.
	listsMigration := func(db *gorm.DB) error {
		return db.Exec(`CREATE TABLE IF NOT EXISTS lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)`).Error
	}
	db := database.NewInMemoryTenantDB(t, listsMigration, buddy.Migration)
	tidA, tidB := uuid.New(), uuid.New()
	listAId, listBId := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&Entity{Id: listAId, TenantId: tidA, CharacterId: 7, Capacity: 20}).Error)
	require.NoError(t, db.Create(&Entity{Id: listBId, TenantId: tidB, CharacterId: 7, Capacity: 50}).Error)
	return db, tidA, tidB, listAId, listBId
}

func TestListProvider_ByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, _, _ := newListsDB(t)

	gotA, err := byCharacterIdEntityProvider(7)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, byte(20), gotA.Capacity, "should be tenant A's row")

	gotB, err := byCharacterIdEntityProvider(7)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantId)
	assert.Equal(t, byte(50), gotB.Capacity, "should be tenant B's row")
}

func TestListAdministrator_UpdateCapacity_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, _, _ := newListsDB(t)

	// updateCapacity calls byCharacterIdEntityProvider then db.Save. Both pieces
	// run under the tenant context so the read must only return tenant A's row,
	// and the save must only update tenant A's row.
	err := updateCapacity(db.WithContext(database.TenantContext(tidA)), 7, 99)
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, byte(99), r.Capacity, "tenant A's capacity should be updated")
		} else {
			assert.Equal(t, byte(50), r.Capacity, "tenant B must be untouched")
		}
	}
	_ = tidB
}
