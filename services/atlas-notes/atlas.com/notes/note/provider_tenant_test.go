package note

import (
	"testing"
	"time"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newNotesDB seeds two note rows in two tenants that overlap on
// (CharacterID, SenderID). Autoincrement primary keys are globally unique
// under sqlite, so the two rows use ids 1 and 2.
func newNotesDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: 1, TenantId: tidA, CharacterID: 1001, SenderID: 2001,
		Message: "tenantA", Timestamp: now, Flag: 0,
	}).Error)
	require.NoError(t, db.Create(&Entity{
		ID: 2, TenantId: tidB, CharacterID: 1001, SenderID: 2001,
		Message: "tenantB", Timestamp: now, Flag: 0,
	}).Error)
	return db, tidA, tidB
}

func TestNoteProvider_GetByCharacterId_FiltersByTenant(t *testing.T) {
	db, tidA, tidB := newNotesDB(t)

	gotA, err := getByCharacterIdProvider(1001)(db.WithContext(databasetest.TenantContext(tidA)))()
	require.NoError(t, err)
	require.Len(t, gotA, 1)
	assert.Equal(t, tidA, gotA[0].TenantId)
	assert.Equal(t, uint32(1), gotA[0].ID)

	gotB, err := getByCharacterIdProvider(1001)(db.WithContext(databasetest.TenantContext(tidB)))()
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, tidB, gotB[0].TenantId)
	assert.Equal(t, uint32(2), gotB[0].ID)
}

func TestNoteAdministrator_UpdateNote_ScopedToTenant(t *testing.T) {
	db, tidA, tidB := newNotesDB(t)

	// updateNote calls tx.Where("id = ?", note.Id()).Updates(&entity) — the
	// tenant callback must keep tenant B's id=2 row untouched.
	modelA, err := NewBuilder().
		SetId(1).
		SetCharacterId(1001).
		SetSenderId(2001).
		SetMessage("tenantA-only").
		SetTimestamp(time.Now()).
		SetFlag(0).
		Build()
	require.NoError(t, err)

	_, err = updateNote(db.WithContext(databasetest.TenantContext(tidA)), tidA, modelA)
	require.NoError(t, err)

	var rows []Entity
	require.NoError(t, db.Unscoped().Order("tenant_id").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		switch r.TenantId {
		case tidA:
			assert.Equal(t, "tenantA-only", r.Message, "tenant A's note should be updated")
		case tidB:
			assert.Equal(t, "tenantB", r.Message, "tenant B must be untouched")
		}
	}
}
