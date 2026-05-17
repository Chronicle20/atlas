package key

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newKeysDB seeds two key rows in two tenants that overlap on CharacterId. The
// composite primary key (CharacterId, Key) is globally unique under sqlite, so
// the two rows use different Key values (10 and 20).
func newKeysDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&entity{TenantId: tidA, CharacterId: 1001, Key: 10, Type: 1, Action: 100}).Error)
	require.NoError(t, db.Create(&entity{TenantId: tidB, CharacterId: 1001, Key: 20, Type: 1, Action: 200}).Error)
	return db, tidA, tidB
}

func TestKeyProvider_ByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newKeysDB(t)

	rowsA, err := byCharacterIdEntityProvider(1001)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rowsA, 1, "tenant A must only see its own key row")
	assert.Equal(t, tidA, rowsA[0].TenantId)
	assert.Equal(t, int32(10), rowsA[0].Key)

	rowsB, err := byCharacterIdEntityProvider(1001)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rowsB, 1)
	assert.Equal(t, tidB, rowsB[0].TenantId)
	assert.Equal(t, int32(20), rowsB[0].Key)
}

func TestKeyAdministrator_DeleteByCharacter_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newKeysDB(t)

	// Without tenant scoping, deleting by character_id alone would wipe both
	// tenants' rows. The callback must scope the delete to tenant A.
	require.NoError(t, deleteByCharacter(db.WithContext(database.TenantContext(tidA)), 1001))

	var rows []entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 1, "tenant B's row must survive tenant A's delete")
	assert.Equal(t, tidB, rows[0].TenantId)
	assert.Equal(t, int32(20), rows[0].Key)
}
