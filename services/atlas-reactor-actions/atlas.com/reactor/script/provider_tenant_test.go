package script

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newScriptTenantDB seeds two reactor-script rows in two tenants that share
// the same ReactorID. UUID primary keys are explicit so the postgres default
// never fires under sqlite.
func newScriptTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, MigrateTable)
	tidA, tidB := uuid.New(), uuid.New()
	idA, idB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: idA, TenantID: tidA, ReactorID: "reactor-1",
		Data: `{"reactorId":"reactor-1","hitRules":[],"actRules":[]}`,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: idB, TenantID: tidB, ReactorID: "reactor-1",
		Data: `{"reactorId":"reactor-1","hitRules":[],"actRules":[]}`,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	return db, tidA, tidB, idA, idB
}

func TestScriptProvider_GetByReactorId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB, idA, idB := newScriptTenantDB(t)

	gotA, err := getByReactorIdProvider("reactor-1")(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	assert.Equal(t, tidA, gotA.TenantID)
	assert.Equal(t, idA, gotA.ID)

	gotB, err := getByReactorIdProvider("reactor-1")(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	assert.Equal(t, tidB, gotB.TenantID)
	assert.Equal(t, idB, gotB.ID)
}

func TestScriptAdministrator_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _, idA, _ := newScriptTenantDB(t)

	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("id = ?", idA).
		Update("data", `{"reactorId":"reactor-1","hitRules":[],"actRules":[],"label":"tenantA-only"}`).Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantID == tidA {
			assert.Contains(t, r.Data, "tenantA-only")
		} else {
			assert.NotContains(t, r.Data, "tenantA-only", "tenant B must be untouched")
		}
	}
}
