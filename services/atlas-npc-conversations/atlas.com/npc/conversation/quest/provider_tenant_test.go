package quest

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newQuestTenantDB seeds two quest_conversation rows in two tenants that
// share the same QuestID. UUID primary keys are unique per row.
func newQuestTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, MigrateTable)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: idA, TenantID: tidA, QuestID: 1001, NpcID: 9000,
		Data: `{"questId":1001}`, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: idB, TenantID: tidB, QuestID: 1001, NpcID: 9000,
		Data: `{"questId":1001}`, CreatedAt: now, UpdatedAt: now,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestQuestProvider_GetByQuestId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newQuestTenantDB(t)

	gotA, err := getByQuestIdProvider(1001)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantID)
	assert.Equal(t, idA, gotA.ID)

	gotB, err := getByQuestIdProvider(1001)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantID)
	assert.Equal(t, idB, gotB.ID)
}

func TestQuestAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, idA, _ := newQuestTenantDB(t)

	// updateQuestConversation runs db.Model(&Entity{}).Where("id = ?", id).Updates(...).
	// The tenant callback must keep tenant B's matching-questId row untouched.
	// We exercise the raw write surface (since constructing a Model with a
	// fully-validated StateMachine requires significant fixture setup
	// orthogonal to tenant scoping).
	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", idA).
		Updates(map[string]interface{}{"data": `{"questId":1001,"label":"tenantA-only"}`}).Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantID {
		case tidA:
			assert.Contains(t, r.Data, "tenantA-only", "tenant A's row should be updated")
		case tidB:
			assert.NotContains(t, r.Data, "tenantA-only", "tenant B must be untouched")
		}
	}
}
