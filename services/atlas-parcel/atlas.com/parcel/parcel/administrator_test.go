package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// TestUpdateStatusIfPending covers the reviewer's "Important" finding:
// UpdateStatusIfPending must be a compare-and-swap on status='pending', not
// an unconditional write. A row that is already resolved (e.g. StatusReceived)
// must be left untouched and report 0 rows affected — under the plain,
// unpredicated UpdateStatus this same call would succeed and clobber the
// row regardless of its current status, which is exactly the race
// Processor.resolve relies on this predicate to close.
func TestUpdateStatusIfPending(t *testing.T) {
	t.Run("pending row is updated", func(t *testing.T) {
		db := databasetest.NewInMemoryTenantDB(t, Migration)
		tid := uuid.New()
		id := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)
		ctx := databasetest.TenantContext(tid)

		require.NoError(t, db.WithContext(ctx).Create(&Entity{
			Id: id, WorldId: 0,
			SenderId: 200, SenderAccountId: 1, SenderName: "A",
			RecipientId: 100, RecipientAccountId: 2,
			Status: StatusPending, CreatedAt: now,
			ReceivableAt: now, ExpiresAt: now.Add(ExpiryWindow),
		}).Error)

		rows, err := UpdateStatusIfPending(db.WithContext(ctx))(id, StatusReceived, now)
		require.NoError(t, err)
		assert.EqualValues(t, 1, rows)

		m, err := ById(id)(db.WithContext(ctx))()
		require.NoError(t, err)
		assert.Equal(t, StatusReceived, m.Status())
	})

	t.Run("already-resolved row is left untouched", func(t *testing.T) {
		db := databasetest.NewInMemoryTenantDB(t, Migration)
		tid := uuid.New()
		id := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)
		earlierResolve := now.Add(-time.Minute)
		ctx := databasetest.TenantContext(tid)

		require.NoError(t, db.WithContext(ctx).Create(&Entity{
			Id: id, WorldId: 0,
			SenderId: 200, SenderAccountId: 1, SenderName: "A",
			RecipientId: 100, RecipientAccountId: 2,
			Status: StatusReceived, CreatedAt: now,
			ReceivableAt: now, ExpiresAt: now.Add(ExpiryWindow),
			ResolvedAt: &earlierResolve,
		}).Error)

		rows, err := UpdateStatusIfPending(db.WithContext(ctx))(id, StatusDiscarded, now)
		require.NoError(t, err)
		assert.EqualValues(t, 0, rows)

		m, err := ById(id)(db.WithContext(ctx))()
		require.NoError(t, err)
		assert.Equal(t, StatusReceived, m.Status())
		require.NotNil(t, m.ResolvedAt())
		assert.True(t, m.ResolvedAt().Equal(earlierResolve))
	})
}
