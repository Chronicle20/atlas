package quest

import (
	"testing"
	"time"

	"atlas-quest/quest/progress"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newQuestTenantDB seeds two quest_status rows in two tenants that share the
// same CharacterId+QuestId. Auto-increment IDs are globally unique under
// sqlite so we use 1 and 2.
func newQuestTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration, progress.Migration)
	tidA, tidB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: 1, TenantId: tidA, CharacterId: 1000, QuestId: 42,
		State: 1, StartedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: 2, TenantId: tidB, CharacterId: 1000, QuestId: 42,
		State: 1, StartedAt: now,
	}).Error)
	return db, tidA, tidB
}

func TestQuestProvider_GetByCharacterAndQuest_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newQuestTenantDB(t)

	gotA, err := byCharacterIdAndQuestIdEntityProvider(1000, 42)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantId)
	assert.Equal(t, uint32(1), gotA.ID)

	gotB, err := byCharacterIdAndQuestIdEntityProvider(1000, 42)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantId)
	assert.Equal(t, uint32(2), gotB.ID)
}

func TestQuestAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newQuestTenantDB(t)

	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", 1).
		Update("completed_count", uint32(9999)).Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, uint32(9999), r.CompletedCount)
		} else {
			assert.NotEqual(t, uint32(9999), r.CompletedCount, "tenant B must be untouched")
		}
	}
}
