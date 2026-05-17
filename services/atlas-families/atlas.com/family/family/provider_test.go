package family

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newFamiliesDB seeds two family-member rows in two tenants. The Entity has a
// uniqueIndex on character_id which sqlite enforces globally, so the two rows
// use different CharacterIds (101 and 102). They overlap on SeniorId (50) and
// non-zero DailyRep so a leak across the tenant boundary is observable through
// both read and write paths.
func newFamiliesDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := database.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	senior := uint32(50)
	require.NoError(t, db.Create(&Entity{
		ID: 1, TenantId: tidA, CharacterId: 101, SeniorId: &senior,
		JuniorIds: []uint32{}, Level: 10, World: 0, DailyRep: 25,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: 2, TenantId: tidB, CharacterId: 102, SeniorId: &senior,
		JuniorIds: []uint32{}, Level: 20, World: 0, DailyRep: 75,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error)
	return db, tidA, tidB
}

func TestFamilyProvider_GetBySeniorId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newFamiliesDB(t)

	rowsA, err := GetBySeniorIdProvider(50)(db.WithContext(database.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, rowsA, 1, "senior 50 has juniors in both tenants — only tenant A's row should return")
	assert.Equal(t, tidA, rowsA[0].TenantId)
	assert.Equal(t, uint32(101), rowsA[0].CharacterId)

	rowsB, err := GetBySeniorIdProvider(50)(db.WithContext(database.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, rowsB, 1)
	assert.Equal(t, tidB, rowsB[0].TenantId)
}

func TestFamilyAdministrator_BatchResetDailyRep_ScopedToTenant(t *testing.T) {
	db, tidA, _ := newFamiliesDB(t)
	log := logrus.New()
	log.SetOutput(nullWriter{})

	// BatchResetDailyRep updates every row with daily_rep > 0. Without tenant
	// scoping it would reset both tenants' counters; with scoping only tenant
	// A's row should be touched.
	result, err := BatchResetDailyRep(db.WithContext(database.TenantContext(tidA)), log)()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.AffectedCount, "only tenant A's row should be affected")

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		if r.TenantId == tidA {
			assert.Equal(t, uint32(0), r.DailyRep, "tenant A's daily rep should be reset")
		} else {
			assert.Equal(t, uint32(75), r.DailyRep, "tenant B's daily rep must be untouched")
		}
	}
}

type nullWriter struct{}

func (nullWriter) Write(p []byte) (int, error) { return len(p), nil }
