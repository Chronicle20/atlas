package marriage

import (
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newMarriagesDB seeds two marriage rows in two tenants that overlap on
// (CharacterId1, CharacterId2, Status). The autoincrement primary key is
// globally unique under sqlite, so the two rows use ids 1 and 2.
func newMarriagesDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: 1, TenantId: tidA, CharacterId1: 1001, CharacterId2: 1002,
		Status: StatusProposed, ProposedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: 2, TenantId: tidB, CharacterId1: 1001, CharacterId2: 1002,
		Status: StatusProposed, ProposedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error)
	return db, tidA, tidB
}

func TestMarriageProvider_GetActiveMarriageByCharacter_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newMarriagesDB(t)
	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	gotA, err := GetActiveMarriageByCharacterProvider(db.WithContext(databasetest.TenantContext(tidA)), log)(1001)()
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.Equal(t, tidA, gotA.TenantId())
	assert.Equal(t, uint32(1), gotA.Id())

	gotB, err := GetActiveMarriageByCharacterProvider(db.WithContext(databasetest.TenantContext(tidB)), log)(1001)()
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.Equal(t, tidB, gotB.TenantId())
	assert.Equal(t, uint32(2), gotB.Id())
}

func TestMarriageAdministrator_UpdateMarriage_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newMarriagesDB(t)
	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	// Tenant A loads its own marriage (id=1), transitions it to Engaged, and
	// saves. UpdateMarriage uses db.Save(&entity) which is a primary-key match;
	// the tenant callback must keep tenant B's id=2 untouched.
	engagedAt := time.Now()
	marriageA, err := NewBuilder(1001, 1002, tidA).
		SetId(1).
		SetStatus(StatusEngaged).
		SetProposedAt(time.Now()).
		SetEngagedAt(&engagedAt).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Build()
	require.NoError(t, err)

	_, err = UpdateMarriage(db.WithContext(databasetest.TenantContext(tidA)), log)(marriageA)()
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantId {
		case tidA:
			assert.Equal(t, StatusEngaged, r.Status, "tenant A's marriage should be engaged")
		case tidB:
			assert.Equal(t, StatusProposed, r.Status, "tenant B must be untouched")
		}
	}
}
