package ledger

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func newLedgerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, database.IdempotencyMigration)
}

func TestClaim_FirstClaimSucceeds(t *testing.T) {
	db := newLedgerTestDB(t)
	ctx := databasetest.TenantContext(uuid.New())
	x := uuid.New()

	err := Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42)

	require.NoError(t, err)
}

func TestClaim_ReplayIsRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	ctx := databasetest.TenantContext(uuid.New())
	x := uuid.New()

	require.NoError(t, Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42))

	err := Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42)

	require.ErrorIs(t, err, ErrAlreadyProcessed)
}

func TestClaim_ReplayUnderDifferentCommandTypeIsStillRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	ctx := databasetest.TenantContext(uuid.New())
	x := uuid.New()

	require.NoError(t, Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42))

	// The key is (tenant, transaction), not (tenant, transaction, type) -- a
	// redelivery does not stop being the same click just because a different
	// command type was passed in by mistake or by a bug elsewhere.
	err := Claim(ctx, db, x, "REQUEST_LOCKER_REBATE", 42)

	require.ErrorIs(t, err, ErrAlreadyProcessed)
}

func TestClaim_ADifferentTransactionSucceeds(t *testing.T) {
	db := newLedgerTestDB(t)
	ctx := databasetest.TenantContext(uuid.New())
	x := uuid.New()

	require.NoError(t, Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42))

	err := Claim(ctx, db, uuid.New(), "REQUEST_GIFT_PURCHASE", 42)

	require.NoError(t, err)
}

func TestClaim_TheSameTransactionUnderADifferentTenantSucceeds(t *testing.T) {
	db := newLedgerTestDB(t)
	tenantA := uuid.New()
	x := uuid.New()

	require.NoError(t, Claim(databasetest.TenantContext(tenantA), db, x, "REQUEST_GIFT_PURCHASE", 42))

	err := Claim(databasetest.TenantContext(uuid.New()), db, x, "REQUEST_GIFT_PURCHASE", 42)

	require.NoError(t, err)
}

func TestClaim_TheZeroTransactionIdIsRejectedOutright(t *testing.T) {
	db := newLedgerTestDB(t)
	ctx := databasetest.TenantContext(uuid.New())

	err := Claim(ctx, db, uuid.Nil, "REQUEST_GIFT_PURCHASE", 42)

	require.Error(t, err)
	require.False(t, errors.Is(err, ErrAlreadyProcessed), "the zero id must not be reported as an already-processed replay")
}

// TestClaim_JoinsCallersTransaction proves Claim writes through the handle it
// is given rather than opening its own: a claim made inside a transaction
// that is then rolled back must not survive the rollback. If it did, a
// command whose later work failed and rolled back would be permanently
// blocked from retry by a claim row that outlived the failure.
func TestClaim_JoinsCallersTransaction(t *testing.T) {
	db := newLedgerTestDB(t)
	ctx := databasetest.TenantContext(uuid.New())
	x := uuid.New()

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := Claim(ctx, tx, x, "REQUEST_GIFT_PURCHASE", 42); err != nil {
			return err
		}
		return errors.New("force rollback: simulated failure after the claim")
	})
	require.Error(t, txErr)

	// The rollback must have taken the claim with it, so the same transaction
	// id can be claimed again outside any transaction.
	err := Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42)

	require.NoError(t, err)
}

// TestClaim_LandsInTheSharedIdempotencyTable proves Claim writes through
// database.IdempotencyEntity / idempotency_keys -- the shared table this
// service already migrates and sweeps -- rather than a new, purpose-built
// table.
func TestClaim_LandsInTheSharedIdempotencyTable(t *testing.T) {
	db := newLedgerTestDB(t)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	x := uuid.New()

	require.NoError(t, Claim(ctx, db, x, "REQUEST_GIFT_PURCHASE", 42))

	var row database.IdempotencyEntity
	err := db.WithContext(ctx).First(&row, "key = ?", x.String()).Error
	require.NoError(t, err)
	require.Equal(t, tenantId, row.TenantId)
	require.Equal(t, x.String(), row.Key)
	require.Equal(t, "REQUEST_GIFT_PURCHASE", row.Operation)
}
