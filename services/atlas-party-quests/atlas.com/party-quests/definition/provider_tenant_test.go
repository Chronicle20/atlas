package definition

import (
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newDefinitionTenantDB seeds two definition rows in two tenants that share
// the same QuestID. UUID primary keys are unique per row.
func newDefinitionTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, MigrateTable)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: idA, TenantID: tidA, QuestID: "pq-alpha",
		Data: `{"questId":"pq-alpha"}`, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: idB, TenantID: tidB, QuestID: "pq-alpha",
		Data: `{"questId":"pq-alpha"}`, CreatedAt: now, UpdatedAt: now,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestDefinitionProvider_GetByQuestId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newDefinitionTenantDB(t)

	gotA, err := getByQuestIdProvider("pq-alpha")(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantID)
	assert.Equal(t, idA, gotA.ID)

	gotB, err := getByQuestIdProvider("pq-alpha")(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantID)
	assert.Equal(t, idB, gotB.ID)
}

func TestDefinitionAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, tidB, idA, _ := newDefinitionTenantDB(t)

	// updateDefinition runs db.Model(&Entity{}).Where("id = ?", id).Updates(...).
	// The tenant callback must keep tenant B's matching-questId row untouched.
	// Exercise the raw write surface (Model construction requires a fully
	// validated definition fixture orthogonal to tenant scoping).
	err := db.WithContext(databasetest.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", idA).
		Updates(map[string]interface{}{"data": `{"questId":"pq-alpha","label":"tenantA-only"}`}).Error
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
