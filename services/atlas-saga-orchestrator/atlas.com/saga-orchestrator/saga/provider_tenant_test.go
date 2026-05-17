package saga

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newSagaTenantDB seeds two saga rows in two tenants. UUID primary keys are
// explicit so AutoMigrate's defaults never need to fire under sqlite.
// Note: saga recovery paths intentionally cross tenants (see plan §11), but
// the non-recovery read/write surface is expected to be tenant-scoped via
// the tenant_id GORM callback.
func newSagaTenantDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	txA, txB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		TransactionId: txA, TenantId: tidA, SagaType: "test",
		InitiatedBy: "tenantA", Status: "active",
		SagaData: []byte(`{"foo":"bar"}`), Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		TransactionId: txB, TenantId: tidB, SagaType: "test",
		InitiatedBy: "tenantB", Status: "active",
		SagaData: []byte(`{"foo":"bar"}`), Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	return db, tidA, tidB, txA, txB
}

func TestSagaStore_GetById_FiltersByTenant(t *testing.T) {
	db, tidA, _, txA, txB := newSagaTenantDB(t)

	// Tenant A reads tenant A's saga.
	var gotA Entity
	require.NoError(t,
		db.WithContext(database.TenantContext(tidA)).
			Where("transaction_id = ?", txA).First(&gotA).Error)
	assert.Equal(t, tidA, gotA.TenantId)

	// Tenant A asking for tenant B's saga must miss (non-recovery path).
	var gotMiss Entity
	err := db.WithContext(database.TenantContext(tidA)).
		Where("transaction_id = ?", txB).First(&gotMiss).Error
	assert.Error(t, err, "tenant A must not see tenant B's saga via tenant-scoped read")
}

func TestSagaStore_Update_ScopedToTenant(t *testing.T) {
	db, tidA, _, txA, _ := newSagaTenantDB(t)

	err := db.WithContext(database.TenantContext(tidA)).
		Model(&Entity{}).
		Where("transaction_id = ?", txA).
		Update("initiated_by", "tenantA-only").Error
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, "tenantA-only", r.InitiatedBy)
		} else {
			assert.NotEqual(t, "tenantA-only", r.InitiatedBy, "tenant B must be untouched")
		}
	}
}
