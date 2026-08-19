package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// newParcelTenantDB seeds two parcels for the same recipientId under two
// different tenant ids. UUID PKs are set explicitly.
func newParcelTenantDB(t *testing.T) (db *gorm.DB, tidA, tidB, idA, idB uuid.UUID) {
	t.Helper()
	db = databasetest.NewInMemoryTenantDB(t, Migration)
	tidA, tidB = uuid.New(), uuid.New()
	idA, idB = uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, db.Create(&Entity{
		Id: idA, TenantId: tidA, WorldId: 0,
		SenderId: 200, SenderAccountId: 1, SenderName: "A",
		RecipientId: 100, RecipientAccountId: 2,
		Status: StatusPending, CreatedAt: now,
		ReceivableAt: now, ExpiresAt: now.Add(ExpiryWindow),
	}).Error)
	require.NoError(t, db.Create(&Entity{
		Id: idB, TenantId: tidB, WorldId: 0,
		SenderId: 200, SenderAccountId: 1, SenderName: "B",
		RecipientId: 100, RecipientAccountId: 2,
		Status: StatusPending, CreatedAt: now,
		ReceivableAt: now, ExpiresAt: now.Add(ExpiryWindow),
	}).Error)
	return
}

func TestProviderTenantIsolation(t *testing.T) {
	t.Run("recipient scoped to tenant", func(t *testing.T) {
		db, tidA, _, idA, _ := newParcelTenantDB(t)

		results, err := ByRecipient(100, world.Id(0), StatusPending)(db.WithContext(databasetest.TenantContext(tidA)))()
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, idA, results[0].Id())
	})

	t.Run("sender scoped to tenant", func(t *testing.T) {
		db, _, tidB, _, idB := newParcelTenantDB(t)

		results, err := BySender(200, StatusPending)(db.WithContext(databasetest.TenantContext(tidB)))()
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, idB, results[0].Id())
	})

	t.Run("receivable filter", func(t *testing.T) {
		db := databasetest.NewInMemoryTenantDB(t, Migration)
		tid := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)

		futureId := uuid.New()
		pastId := uuid.New()
		require.NoError(t, db.Create(&Entity{
			Id: futureId, TenantId: tid, WorldId: 0,
			SenderId: 200, SenderAccountId: 1, SenderName: "A",
			RecipientId: 100, RecipientAccountId: 2,
			Status: StatusPending, CreatedAt: now,
			ReceivableAt: now.Add(time.Hour), ExpiresAt: now.Add(ExpiryWindow),
		}).Error)
		require.NoError(t, db.Create(&Entity{
			Id: pastId, TenantId: tid, WorldId: 0,
			SenderId: 200, SenderAccountId: 1, SenderName: "B",
			RecipientId: 100, RecipientAccountId: 2,
			Status: StatusPending, CreatedAt: now,
			ReceivableAt: now.Add(-time.Hour), ExpiresAt: now.Add(ExpiryWindow),
		}).Error)

		results, err := ReceivableByRecipient(100, world.Id(0), now)(db.WithContext(databasetest.TenantContext(tid)))()
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, pastId, results[0].Id())
	})
}
