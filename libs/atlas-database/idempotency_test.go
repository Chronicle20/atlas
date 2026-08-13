package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

type widget struct {
	TenantId uuid.UUID `gorm:"not null"`
	Id       uint32    `gorm:"primaryKey;autoIncrement"`
	Name     string
}

func widgetMigration(db *gorm.DB) error { return db.AutoMigrate(&widget{}) }

type acceptBody struct {
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`
}

func testDB(t *testing.T) (*gorm.DB, context.Context, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, database.IdempotencyMigration, widgetMigration)
	tenantId := uuid.New()
	return db, databasetest.TenantContext(tenantId), tenantId
}

func countWidgets(t *testing.T, db *gorm.DB, ctx context.Context) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.WithContext(ctx).Model(&widget{}).Count(&n).Error)
	return n
}

func TestOnceRunsTheFirstTime(t *testing.T) {
	db, ctx, _ := testDB(t)

	err := database.Once(ctx, db, "key-1", "ACCEPT", func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Create(&widget{Name: "first"}).Error
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), countWidgets(t, db, ctx))
}

func TestOnceSkipsTheSecondDelivery(t *testing.T) {
	db, ctx, _ := testDB(t)
	create := func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Create(&widget{Name: "dup"}).Error
	}

	require.NoError(t, database.Once(ctx, db, "key-1", "ACCEPT", create))
	err := database.Once(ctx, db, "key-1", "ACCEPT", create)

	require.ErrorIs(t, err, database.ErrDuplicate)
	require.Equal(t, int64(1), countWidgets(t, db, ctx), "redelivery must not create a second row")
}

func TestOnceIsScopedPerTenant(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, database.IdempotencyMigration, widgetMigration)
	ctxA := databasetest.TenantContext(uuid.New())
	ctxB := databasetest.TenantContext(uuid.New())

	require.NoError(t, database.Once(ctxA, db, "same-key", "ACCEPT", func(tx *gorm.DB) error {
		return tx.WithContext(ctxA).Create(&widget{Name: "a"}).Error
	}))
	require.NoError(t, database.Once(ctxB, db, "same-key", "ACCEPT", func(tx *gorm.DB) error {
		return tx.WithContext(ctxB).Create(&widget{Name: "b"}).Error
	}))

	require.Equal(t, int64(1), countWidgets(t, db, ctxA))
	require.Equal(t, int64(1), countWidgets(t, db, ctxB))
}

func TestOnceRollsBackTheClaimWhenTheWorkFails(t *testing.T) {
	db, ctx, _ := testDB(t)
	boom := func(tx *gorm.DB) error { return context.DeadlineExceeded }

	err := database.Once(ctx, db, "key-1", "ACCEPT", boom)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// The key must NOT be claimed — a failed apply has to remain retryable.
	err = database.Once(ctx, db, "key-1", "ACCEPT", func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Create(&widget{Name: "retry"}).Error
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), countWidgets(t, db, ctx))
}

func TestOnceRequiresATenant(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, database.IdempotencyMigration, widgetMigration)

	err := database.Once(context.Background(), db, "key-1", "ACCEPT", func(tx *gorm.DB) error {
		return nil
	})

	require.Error(t, err)
	require.NotErrorIs(t, err, database.ErrDuplicate)
}

func TestKeyIsStableForIdenticalPayloads(t *testing.T) {
	tid := uuid.New()
	body := acceptBody{TemplateId: 1812000, Quantity: 1}

	a, err := database.Key(tid, "ACCEPT", body)
	require.NoError(t, err)
	b, err := database.Key(tid, "ACCEPT", acceptBody{TemplateId: 1812000, Quantity: 1})
	require.NoError(t, err)

	require.Equal(t, a, b)
	require.NotEmpty(t, a)
}

func TestKeyVariesByTransactionOperationAndPayload(t *testing.T) {
	tid := uuid.New()
	other := uuid.New()
	body := acceptBody{TemplateId: 1812000, Quantity: 1}

	base, err := database.Key(tid, "ACCEPT", body)
	require.NoError(t, err)

	byTransaction, err := database.Key(other, "ACCEPT", body)
	require.NoError(t, err)
	byOperation, err := database.Key(tid, "RELEASE", body)
	require.NoError(t, err)
	byPayload, err := database.Key(tid, "ACCEPT", acceptBody{TemplateId: 1812001, Quantity: 1})
	require.NoError(t, err)

	require.NotEqual(t, base, byTransaction)
	require.NotEqual(t, base, byOperation)
	require.NotEqual(t, base, byPayload)
}

func TestApplyOnceSwallowsTheDuplicate(t *testing.T) {
	db, ctx, _ := testDB(t)
	l, _ := test.NewNullLogger()
	tid := uuid.New()
	body := acceptBody{TemplateId: 1812000, Quantity: 1}
	create := func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Create(&widget{Name: "meso magnet"}).Error
	}

	require.NoError(t, database.ApplyOnce(l, ctx, db, tid, "ACCEPT", body, create))
	// Same transaction, same operation, same body — a redelivery.
	require.NoError(t, database.ApplyOnce(l, ctx, db, tid, "ACCEPT", body, create),
		"a duplicate delivery must not surface as an error")

	require.Equal(t, int64(1), countWidgets(t, db, ctx))
}

func TestApplyOnceStillAppliesDistinctCommands(t *testing.T) {
	db, ctx, _ := testDB(t)
	l, _ := test.NewNullLogger()
	tid := uuid.New()
	create := func(tx *gorm.DB) error {
		return tx.WithContext(ctx).Create(&widget{Name: "w"}).Error
	}

	require.NoError(t, database.ApplyOnce(l, ctx, db, tid, "ACCEPT", acceptBody{TemplateId: 1812000}, create))
	require.NoError(t, database.ApplyOnce(l, ctx, db, tid, "ACCEPT", acceptBody{TemplateId: 1812001}, create))
	require.NoError(t, database.ApplyOnce(l, ctx, db, tid, "RELEASE", acceptBody{TemplateId: 1812000}, create))

	require.Equal(t, int64(3), countWidgets(t, db, ctx))
}

func TestApplyOncePropagatesWorkFailures(t *testing.T) {
	db, ctx, _ := testDB(t)
	l, _ := test.NewNullLogger()

	err := database.ApplyOnce(l, ctx, db, uuid.New(), "ACCEPT", acceptBody{}, func(tx *gorm.DB) error {
		return context.DeadlineExceeded
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSweepDeletesOnlyExpiredKeys(t *testing.T) {
	db, ctx, _ := testDB(t)

	require.NoError(t, database.Once(ctx, db, "old", "ACCEPT", func(tx *gorm.DB) error { return nil }))
	require.NoError(t, database.Once(ctx, db, "fresh", "ACCEPT", func(tx *gorm.DB) error { return nil }))
	require.NoError(t, db.WithContext(ctx).Model(&database.IdempotencyEntity{}).
		Where("key = ?", "old").
		Update("created_at", time.Now().Add(-48*time.Hour)).Error)

	require.NoError(t, database.SweepIdempotency(ctx, db, 24*time.Hour))

	var remaining []database.IdempotencyEntity
	require.NoError(t, db.WithContext(ctx).Find(&remaining).Error)
	require.Len(t, remaining, 1)
	require.Equal(t, "fresh", remaining[0].Key)
}
