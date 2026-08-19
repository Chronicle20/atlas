package equipslot

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Uniquely-named shared-cache in-memory database, mirroring
	// teleport_rock/administrator_test.go: a bare ":memory:" DB is private
	// to one connection, so a second pooled connection can see an empty
	// schema. Shared-cache is visible to every pooled connection; the
	// unique name keeps each test isolated.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0)
		sqlDB.SetConnMaxIdleTime(0)
	}
	require.NoError(t, Migration(db))
	return db
}

// S is the Atlas canonical equipped-inventory position for the extended
// pendant slot (see derivation-equip-slot.md E1 / R1); it comes from the
// shared constants table, not from a bare literal.
func testSlotIndex(t *testing.T) int16 {
	t.Helper()
	s, err := slot.GetSlotByType("pendant2")
	require.NoError(t, err)
	return int16(s.Position)
}

func assertWithinTolerance(t *testing.T, expected time.Time, actual time.Time, tolerance time.Duration) {
	t.Helper()
	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqualf(t, diff, tolerance, "expected [%s] to be within [%s] of [%s]", actual, tolerance, expected)
}

func TestExtend(t *testing.T) {
	tenantA := uuid.New()
	S := testSlotIndex(t)
	tolerance := time.Minute

	t.Run("first purchase creates", func(t *testing.T) {
		db := testDB(t)
		characterId := uint32(42)

		expiresAt, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, uuid.Nil)
		require.NoError(t, err)
		assertWithinTolerance(t, time.Now().Add(30*24*time.Hour), expiresAt, tolerance)

		active, err := GetActive(db, tenantA, characterId)
		require.NoError(t, err)
		require.Len(t, active, 1)
		assert.Equal(t, S, active[0].SlotIndex())
	})

	t.Run("second purchase extends, not duplicates", func(t *testing.T) {
		db := testDB(t)
		characterId := uint32(42)

		_, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, uuid.Nil)
		require.NoError(t, err)

		expiresAt, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, uuid.Nil)
		require.NoError(t, err)
		assertWithinTolerance(t, time.Now().Add(60*24*time.Hour), expiresAt, tolerance)

		active, err := GetActive(db, tenantA, characterId)
		require.NoError(t, err)
		require.Len(t, active, 1)
	})

	t.Run("an expired extension restarts from now", func(t *testing.T) {
		db := testDB(t)
		characterId := uint32(42)

		expired := &Entity{
			Id:          uuid.New(),
			TenantId:    tenantA,
			CharacterId: characterId,
			SlotIndex:   S,
			ExpiresAt:   time.Now().Add(-10 * 24 * time.Hour),
		}
		require.NoError(t, db.Create(expired).Error)

		expiresAt, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, uuid.Nil)
		require.NoError(t, err)
		assertWithinTolerance(t, time.Now().Add(30*24*time.Hour), expiresAt, tolerance)
	})

	t.Run("an expired extension is not active", func(t *testing.T) {
		db := testDB(t)
		characterId := uint32(42)

		expired := &Entity{
			Id:          uuid.New(),
			TenantId:    tenantA,
			CharacterId: characterId,
			SlotIndex:   S,
			ExpiresAt:   time.Now().Add(-10 * 24 * time.Hour),
		}
		require.NoError(t, db.Create(expired).Error)

		active, err := GetActive(db, tenantA, characterId)
		require.NoError(t, err)
		assert.Empty(t, active)
	})

	t.Run("another character is separate", func(t *testing.T) {
		db := testDB(t)
		_, err := Extend(db, tenantA, 42, S, 30*24*time.Hour, uuid.Nil)
		require.NoError(t, err)

		active, err := GetActive(db, tenantA, 99)
		require.NoError(t, err)
		assert.Empty(t, active)
	})

	t.Run("another tenant is separate", func(t *testing.T) {
		db := testDB(t)
		_, err := Extend(db, tenantA, 42, S, 30*24*time.Hour, uuid.Nil)
		require.NoError(t, err)

		active, err := GetActive(db, uuid.New(), 42)
		require.NoError(t, err)
		assert.Empty(t, active)
	})

	t.Run("a repeated transaction id does not double-extend", func(t *testing.T) {
		db := testDB(t)
		characterId := uint32(42)
		txId := uuid.New()

		first, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, txId)
		require.NoError(t, err)

		second, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, txId)
		require.NoError(t, err)
		assert.True(t, first.Equal(second), "a redelivered call for the SAME transaction id must not add days again: first [%s], second [%s]", first, second)

		active, err := GetActive(db, tenantA, characterId)
		require.NoError(t, err)
		require.Len(t, active, 1)
		assertWithinTolerance(t, time.Now().Add(30*24*time.Hour), active[0].ExpiresAt(), tolerance)
	})

	t.Run("a genuinely new transaction id still extends", func(t *testing.T) {
		db := testDB(t)
		characterId := uint32(42)

		_, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, uuid.New())
		require.NoError(t, err)

		expiresAt, err := Extend(db, tenantA, characterId, S, 30*24*time.Hour, uuid.New())
		require.NoError(t, err)
		assertWithinTolerance(t, time.Now().Add(60*24*time.Hour), expiresAt, tolerance)
	})
}
