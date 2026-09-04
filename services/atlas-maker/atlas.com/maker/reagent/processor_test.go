package reagent_test

import (
	"atlas-maker/reagent"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func seedReagent(t *testing.T, db *gorm.DB, tenantId uuid.UUID, itemId item.Id, stat string, value int16) {
	t.Helper()
	m, err := reagent.NewBuilder(tenantId, itemId).
		SetStat(stat).
		SetValue(value).
		Build()
	require.NoError(t, err)
	require.NoError(t, reagent.CreateReagent(db, m))
}

// TestGetByItemIdReturnsSeededReagent seeds row 1 of the derived table
// (0425.img/04250000/info/incPAD = 1) and reads it back through the processor.
func TestGetByItemIdReturnsSeededReagent(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	seedReagent(t, db, tenantId, item.Id(4250000), "incPAD", 1)

	m, err := reagent.NewProcessor(testLogger(), ctx, db).GetByItemId(item.Id(4250000))
	require.NoError(t, err)

	assert.Equal(t, item.Id(4250000), m.ReagentItemId())
	assert.Equal(t, "incPAD", m.Stat())
	assert.Equal(t, int16(1), m.Value())
}

// TestGetByItemIdReturnsNegativeValue pins the signed column: incReqLevel is
// negative for all three of its derived rows.
func TestGetByItemIdReturnsNegativeValue(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	seedReagent(t, db, tenantId, item.Id(4251202), "incReqLevel", -3)

	m, err := reagent.NewProcessor(testLogger(), ctx, db).GetByItemId(item.Id(4251202))
	require.NoError(t, err)
	assert.Equal(t, int16(-3), m.Value())
}

// TestGetByItemIdIsTenantScoped is PRD §8's multi-tenancy requirement made
// executable: the same reagent id retuned differently under two tenants must
// read back per tenant, and neither context may see the other's row.
func TestGetByItemIdIsTenantScoped(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantA := uuid.New()
	tenantB := uuid.New()
	seedReagent(t, db, tenantA, item.Id(4250002), "incPAD", 3)
	seedReagent(t, db, tenantB, item.Id(4250002), "incPAD", 30)

	mA, err := reagent.NewProcessor(testLogger(), databasetest.TenantContext(tenantA), db).GetByItemId(item.Id(4250002))
	require.NoError(t, err)
	assert.Equal(t, int16(3), mA.Value())
	assert.Equal(t, tenantA, mA.TenantId())

	mB, err := reagent.NewProcessor(testLogger(), databasetest.TenantContext(tenantB), db).GetByItemId(item.Id(4250002))
	require.NoError(t, err)
	assert.Equal(t, int16(30), mB.Value())
	assert.Equal(t, tenantB, mB.TenantId())
}

// TestGetByItemIdNotFound pins the distinguishable not-found error Task 23
// relies on to drop an unheld reagent rather than fail the craft (FR-3.2).
func TestGetByItemIdNotFound(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	seedReagent(t, db, tenantId, item.Id(4250000), "incPAD", 1)

	_, err := reagent.NewProcessor(testLogger(), ctx, db).GetByItemId(item.Id(2000000))
	require.Error(t, err)
	assert.ErrorIs(t, err, reagent.ErrNotFound)
}

// TestGetByItemIdNotFoundAcrossTenants asserts the not-found path is also what
// a tenant sees for a row that exists only under another tenant.
func TestGetByItemIdNotFoundAcrossTenants(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantA := uuid.New()
	tenantB := uuid.New()
	seedReagent(t, db, tenantA, item.Id(4250100), "incMAD", 1)

	_, err := reagent.NewProcessor(testLogger(), databasetest.TenantContext(tenantB), db).GetByItemId(item.Id(4250100))
	assert.ErrorIs(t, err, reagent.ErrNotFound)
}

func TestGetAllReturnsEveryReagentForTheTenant(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantA := uuid.New()
	tenantB := uuid.New()
	seedReagent(t, db, tenantA, item.Id(4250000), "incPAD", 1)
	seedReagent(t, db, tenantA, item.Id(4250601), "incMaxHP", 20)
	seedReagent(t, db, tenantB, item.Id(4251300), "randOption", 1)

	ms, err := reagent.NewProcessor(testLogger(), databasetest.TenantContext(tenantA), db).GetAll()
	require.NoError(t, err)
	require.Len(t, ms, 2)

	byId := make(map[item.Id]reagent.Model, len(ms))
	for _, m := range ms {
		byId[m.ReagentItemId()] = m
	}
	require.Contains(t, byId, item.Id(4250000))
	require.Contains(t, byId, item.Id(4250601))
	assert.Equal(t, "incMaxHP", byId[item.Id(4250601)].Stat())
	assert.Equal(t, int16(20), byId[item.Id(4250601)].Value())
}
