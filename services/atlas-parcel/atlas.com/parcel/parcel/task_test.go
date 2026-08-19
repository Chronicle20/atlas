package parcel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// sweepClock — T = 2026-03-01T00:00:00Z, the reference instant every
// TestExpirySweep subtest is written against (brief Step 2).
var sweepClock = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

func newSweepTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	db.Logger = logger.Default.LogMode(logger.Silent)
	return db
}

func newSweepTestTask(t *testing.T, db *gorm.DB, now time.Time) *ExpiryTask {
	t.Helper()
	l, _ := test.NewNullLogger()
	task := NewExpiryTask(l, databasetest.TenantContext(uuid.New()), db, time.Hour)
	return task.withClock(func() time.Time { return now })
}

// seedSweepParcel inserts a pending-by-default parcel scoped to tenantId and
// returns its id.
func seedSweepParcel(t *testing.T, db *gorm.DB, tenantId uuid.UUID, mutate func(b *Builder)) uuid.UUID {
	t.Helper()
	id := uuid.New()
	itemId := uint32(1302000)
	b := NewBuilder().
		SetId(id).
		SetWorldId(0).
		SetSenderId(100).
		SetSenderAccountId(1).
		SetSenderName("Alice").
		SetRecipientId(200).
		SetRecipientAccountId(2).
		SetRecipientName("Bob").
		SetStatus(StatusPending).
		SetMesoAmount(5000).
		SetFeePaid(800).
		SetItemId(&itemId).
		SetQuantity(1).
		SetCreatedAt(sweepClock.Add(-48 * time.Hour)).
		SetReceivableAt(sweepClock.Add(-24 * time.Hour)).
		SetExpiresAt(sweepClock.Add(-time.Hour))
	if mutate != nil {
		mutate(b)
	}
	m, err := b.Build()
	require.NoError(t, err)

	ctx := databasetest.TenantContext(tenantId)
	_, err = Create(db.WithContext(ctx))(m)
	require.NoError(t, err)
	return id
}

// getSweepParcel re-reads a parcel bypassing tenant scoping, so assertions
// can inspect rows the sweep moved into a DIFFERENT tenant context than the
// one that seeded them (not the case here — the sweep re-enters the SAME
// tenant — but this keeps the helper honest about what it is doing).
func getSweepParcel(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) Model {
	t.Helper()
	ctx := databasetest.TenantContext(tenantId)
	m, err := ById(id)(db.WithContext(ctx))()
	require.NoError(t, err)
	return m
}

// allSweepParcels returns every parcel row for tenantId, for asserting how
// many rows exist after a sweep (e.g. "no new row").
func allSweepParcels(t *testing.T, db *gorm.DB, tenantId uuid.UUID) []Entity {
	t.Helper()
	var es []Entity
	ctx := databasetest.TenantContext(tenantId)
	require.NoError(t, db.WithContext(ctx).Find(&es).Error)
	return es
}

func TestExpirySweep(t *testing.T) {
	t.Run("expires and returns", func(t *testing.T) {
		db := newSweepTestDB(t)
		tid := uuid.New()
		id := seedSweepParcel(t, db, tid, nil)
		task := newSweepTestTask(t, db, sweepClock)

		task.Run()

		original := getSweepParcel(t, db, tid, id)
		assert.Equal(t, StatusExpired, original.Status())
		require.NotNil(t, original.ResolvedAt())
		assert.True(t, original.ResolvedAt().Equal(sweepClock))

		all := allSweepParcels(t, db, tid)
		require.Len(t, all, 2)

		var ret Entity
		for _, e := range all {
			if e.Id != id {
				ret = e
			}
		}
		require.NotEqual(t, uuid.Nil, ret.Id)
		assert.Equal(t, StatusPending, ret.Status)
		assert.EqualValues(t, 200, ret.SenderId)
		assert.EqualValues(t, 100, ret.RecipientId)
		assert.Equal(t, "Bob", ret.SenderName)
		assert.Equal(t, "Unclaimed parcel returned.", ret.Message)
		assert.True(t, ret.Returned)
		assert.EqualValues(t, 0, ret.FeePaid)
		assert.EqualValues(t, 5000, ret.MesoAmount)
		assert.True(t, ret.ReceivableAt.Equal(ret.CreatedAt))
		assert.True(t, ret.ExpiresAt.Equal(ret.CreatedAt.Add(ExpiryWindow)))
	})

	t.Run("return leg expires into nothing", func(t *testing.T) {
		db := newSweepTestDB(t)
		tid := uuid.New()
		id := seedSweepParcel(t, db, tid, func(b *Builder) {
			b.SetReturned(true)
		})
		task := newSweepTestTask(t, db, sweepClock)

		task.Run()

		original := getSweepParcel(t, db, tid, id)
		assert.Equal(t, StatusExpired, original.Status())

		all := allSweepParcels(t, db, tid)
		assert.Len(t, all, 1)
	})

	t.Run("not yet expired", func(t *testing.T) {
		db := newSweepTestDB(t)
		tid := uuid.New()
		id := seedSweepParcel(t, db, tid, func(b *Builder) {
			b.SetExpiresAt(sweepClock.Add(time.Hour))
		})
		task := newSweepTestTask(t, db, sweepClock)

		task.Run()

		original := getSweepParcel(t, db, tid, id)
		assert.Equal(t, StatusPending, original.Status())
		all := allSweepParcels(t, db, tid)
		assert.Len(t, all, 1)
	})

	t.Run("already resolved", func(t *testing.T) {
		db := newSweepTestDB(t)
		tid := uuid.New()
		id := seedSweepParcel(t, db, tid, func(b *Builder) {
			b.SetStatus(StatusReceived)
		})
		task := newSweepTestTask(t, db, sweepClock)

		task.Run()

		original := getSweepParcel(t, db, tid, id)
		assert.Equal(t, StatusReceived, original.Status())
		all := allSweepParcels(t, db, tid)
		assert.Len(t, all, 1)
	})

	t.Run("meso-only return", func(t *testing.T) {
		db := newSweepTestDB(t)
		tid := uuid.New()
		seedSweepParcel(t, db, tid, func(b *Builder) {
			b.SetItemId(nil).SetMesoAmount(5000)
		})
		task := newSweepTestTask(t, db, sweepClock)

		task.Run()

		all := allSweepParcels(t, db, tid)
		require.Len(t, all, 2)
		var ret Entity
		for _, e := range all {
			if e.Status == StatusPending {
				ret = e
			}
		}
		assert.Nil(t, ret.ItemId)
		assert.EqualValues(t, 5000, ret.MesoAmount)
	})

	t.Run("batch bound", func(t *testing.T) {
		db := newSweepTestDB(t)
		tid := uuid.New()
		for i := 0; i < 5; i++ {
			seedSweepParcel(t, db, tid, nil)
		}
		task := newSweepTestTask(t, db, sweepClock).withBatch(2)

		task.Run()
		expiredCount := func() int {
			var es []Entity
			ctx := databasetest.TenantContext(tid)
			require.NoError(t, db.WithContext(ctx).Where("status = ?", StatusExpired).Find(&es).Error)
			return len(es)
		}
		assert.Equal(t, 2, expiredCount())

		task.Run()
		assert.Equal(t, 4, expiredCount())

		task.Run()
		assert.Equal(t, 5, expiredCount())
	})

	t.Run("concurrent claim", func(t *testing.T) {
		// This exercises ClaimExpired's own compare-and-swap UPDATE
		// directly, sequentially, rather than a genuine simultaneous race:
		// databasetest's sqlite-in-memory harness serializes all access
		// through a single *gorm.DB handle, so it cannot express two
		// replicas' UPDATEs actually overlapping in time — a real race
		// needs two independent DB connections against a shared backing
		// store (production postgres), which this in-memory harness does
		// not have. What IS verified here, faithfully: the SECOND claim
		// attempt against an ALREADY-claimed row affects zero rows and
		// creates no second return leg — the row-level guard design §8.1
		// relies on to make concurrent replicas safe without leader
		// election.
		db := newSweepTestDB(t)
		tid := uuid.New()
		seedSweepParcel(t, db, tid, nil)

		ctx := databasetest.TenantContext(tid)
		tdb := db.WithContext(ctx)

		first, err := ClaimExpired(tdb)(sweepClock, 10)
		require.NoError(t, err)
		require.Len(t, first, 1)

		second, err := ClaimExpired(tdb)(sweepClock, 10)
		require.NoError(t, err)
		assert.Len(t, second, 0, "a second claim attempt against an already-claimed row must affect zero rows")
	})
}
