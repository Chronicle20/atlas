package coupon

// NOTE ON THE HARNESS AND WHAT IT DOES / DOES NOT PROVE.
//
// These tests run against gorm's SQLite in-memory driver via
// databasetest.NewInMemoryTenantDB, NOT Postgres. A human ruling on this
// branch selected SQLite in-memory as the harness for this plan's DB tests
// (testcontainers Postgres was available and deliberately declined).
//
// What this harness CAN verify, and does:
//   - that reserveUse's conditional UPDATE carries the right WHERE clause, and
//     that RowsAffected is 1 on a successful claim and 0 on a refusal — GORM
//     reports RowsAffected on SQLite, so the verdict is a real verdict;
//   - that max_uses IS NULL means unlimited;
//   - that the statement is tenant-scoped;
//   - that releaseUse never underflows the unsigned redemption_count column.
//
// What this harness CANNOT verify:
//   - true concurrency. SQLite in-memory is capped to a single connection and
//     serializes writers, so nothing here demonstrates that two SIMULTANEOUS
//     redemptions of a max_uses = 1 coupon resolve to exactly one success.
//     The single-statement check-and-increment is what makes that safe under
//     Postgres, but these tests are not evidence of it. Do not read a passing
//     run as proof of race-safety.

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newCouponTestDB(t *testing.T) (*gorm.DB, tenant.Model) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return db, tm
}

func otherTenantModel(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tm
}

func seedCoupon(t *testing.T, db *gorm.DB, tm tenant.Model, b *Builder) uuid.UUID {
	t.Helper()
	m, err := b.Build()
	require.NoError(t, err)
	created, err := CreateEntity(db, tm, m)
	require.NoError(t, err)
	return created.Id()
}

func loadCount(t *testing.T, db *gorm.DB, id uuid.UUID) uint32 {
	t.Helper()
	var e Entity
	require.NoError(t, db.Where("id = ?", id).First(&e).Error)
	return e.RedemptionCount
}

func TestReserveUseRespectsMaxUses(t *testing.T) {
	db, tm := newCouponTestDB(t)
	id := seedCoupon(t, db, tm, NewBuilder("LIMITED").SetMaxUses(ptrU32(2)).SetRewards(Rewards{NewCurrencyReward(1, 1)}))

	for i := 1; i <= 2; i++ {
		ok, err := reserveUse(db, tm, id)
		if err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("reserve %d = false, want true", i)
		}
	}
	ok, err := reserveUse(db, tm, id)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("third reserve succeeded, want refusal at maxUses")
	}
	if got := loadCount(t, db, id); got != 2 {
		t.Errorf("redemptionCount = %d, want 2", got)
	}
}

func TestReserveUseUnlimitedWhenMaxUsesIsNull(t *testing.T) {
	db, tm := newCouponTestDB(t)
	id := seedCoupon(t, db, tm, NewBuilder("UNLIMITED").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	for i := 0; i < 5; i++ {
		if ok, err := reserveUse(db, tm, id); err != nil || !ok {
			t.Fatalf("reserve %d: ok=%v err=%v", i, ok, err)
		}
	}
	if got := loadCount(t, db, id); got != 5 {
		t.Errorf("redemptionCount = %d, want 5", got)
	}
}

func TestReserveUseIsTenantScoped(t *testing.T) {
	db, tm := newCouponTestDB(t)
	other := otherTenantModel(t)
	id := seedCoupon(t, db, tm, NewBuilder("SCOPED").SetMaxUses(ptrU32(1)).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	ok, err := reserveUse(db, other, id)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("another tenant reserved this coupon")
	}
}

func TestReleaseUseDecrementsWithoutGoingNegative(t *testing.T) {
	db, tm := newCouponTestDB(t)
	id := seedCoupon(t, db, tm, NewBuilder("REL").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	if _, err := reserveUse(db, tm, id); err != nil {
		t.Fatal(err)
	}
	if err := releaseUse(db, tm, id); err != nil {
		t.Fatal(err)
	}
	if got := loadCount(t, db, id); got != 0 {
		t.Errorf("redemptionCount = %d, want 0", got)
	}
	// A stray release must not underflow the unsigned column.
	if err := releaseUse(db, tm, id); err != nil {
		t.Fatal(err)
	}
	if got := loadCount(t, db, id); got != 0 {
		t.Errorf("redemptionCount = %d after a stray release, want 0", got)
	}
}

// redemptionRow is a TEST-ONLY shadow of coupon_redemptions, carrying only the
// two columns deleteEntity reads. Package coupon/redemption imports package
// coupon, so an internal test in this package cannot import it back to get the
// real Entity — that would be an import cycle.
type redemptionRow struct {
	Id       uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId uuid.UUID `gorm:"not null"`
	CouponId uuid.UUID `gorm:"not null;type:uuid"`
}

func (redemptionRow) TableName() string { return redemptionsTable }

func TestDeleteEntityRefusesWhenRedemptionsExist(t *testing.T) {
	db, tm := newCouponTestDB(t)
	require.NoError(t, db.AutoMigrate(&redemptionRow{}))
	id := seedCoupon(t, db, tm, NewBuilder("DELME").SetRewards(Rewards{NewCurrencyReward(1, 1)}))

	require.NoError(t, db.Create(&redemptionRow{Id: uuid.New(), TenantId: tm.Id(), CouponId: id}).Error)

	err := deleteEntity(db, tm, id)
	assert.True(t, errors.Is(err, ErrHasRedemptions), "want ErrHasRedemptions, got %v", err)

	var remaining int64
	require.NoError(t, db.Model(&Entity{}).Where("id = ?", id).Count(&remaining).Error)
	assert.Equal(t, int64(1), remaining, "the coupon must survive a refused delete")
}

func TestDeleteEntitySucceedsWithoutRedemptionsAndIsTenantScoped(t *testing.T) {
	db, tm := newCouponTestDB(t)
	require.NoError(t, db.AutoMigrate(&redemptionRow{}))
	other := otherTenantModel(t)
	id := seedCoupon(t, db, tm, NewBuilder("DELOK").SetRewards(Rewards{NewCurrencyReward(1, 1)}))

	// Another tenant cannot delete this coupon.
	assert.ErrorIs(t, deleteEntity(db, other, id), gorm.ErrRecordNotFound)
	var remaining int64
	require.NoError(t, db.Model(&Entity{}).Where("id = ?", id).Count(&remaining).Error)
	require.Equal(t, int64(1), remaining)

	require.NoError(t, deleteEntity(db, tm, id))
	require.NoError(t, db.Model(&Entity{}).Where("id = ?", id).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)
}

func TestUpdateEntityNeverClobbersRedemptionCountAndIsTenantScoped(t *testing.T) {
	db, tm := newCouponTestDB(t)
	other := otherTenantModel(t)
	id := seedCoupon(t, db, tm, NewBuilder("UPD").SetDescription("before").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	require.NoError(t, db.Model(&Entity{}).Where("id = ?", id).UpdateColumn("redemption_count", 3).Error)

	m, err := NewBuilder("UPD").SetId(id).SetDescription("after").SetActive(false).SetRewards(Rewards{NewCurrencyReward(2, 7)}).Build()
	require.NoError(t, err)

	// Another tenant's update must find nothing to write.
	_, err = updateEntity(db, other, m)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	got, err := updateEntity(db, tm, m)
	require.NoError(t, err)
	assert.Equal(t, "after", got.Description())
	assert.False(t, got.Active())
	assert.Equal(t, uint32(3), got.RedemptionCount(),
		"updateEntity must not write redemption_count — that column belongs to reserveUse/releaseUse")
}

func TestProvidersAreTenantScopedAndFilterOnlyWhatIsSet(t *testing.T) {
	db, tm := newCouponTestDB(t)
	other := otherTenantModel(t)

	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)
	batchId := uuid.New()

	activeId := seedCoupon(t, db, tm, NewBuilder("ACTIVE").SetExpiresAt(&future).SetBatchId(batchId).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	inactiveId := seedCoupon(t, db, tm, NewBuilder("INACTIVE").SetActive(false).SetExpiresAt(&past).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	// Same code in another tenant — a leak across the boundary is observable.
	seedCoupon(t, db, other, NewBuilder("ACTIVE").SetRewards(Rewards{NewCurrencyReward(1, 1)}))

	t.Run("byCode is tenant scoped", func(t *testing.T) {
		got, err := byCodeEntityProvider(tm, "ACTIVE")(db)()
		require.NoError(t, err)
		assert.Equal(t, activeId, got.Id)

		_, err = byCodeEntityProvider(otherTenantModel(t), "ACTIVE")(db)()
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("byId is tenant scoped", func(t *testing.T) {
		got, err := byIdEntityProvider(tm, activeId)(db)()
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", got.Code)

		_, err = byIdEntityProvider(other, activeId)(db)()
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("empty filters list only this tenant", func(t *testing.T) {
		got, err := allEntityProvider(tm, Filters{})(db)()
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	inactive := false
	code := "INACTIVE"
	for name, tc := range map[string]struct {
		f    Filters
		want []uuid.UUID
	}{
		"active":        {Filters{Active: &inactive}, []uuid.UUID{inactiveId}},
		"code":          {Filters{Code: &code}, []uuid.UUID{inactiveId}},
		"batchId":       {Filters{BatchId: &batchId}, []uuid.UUID{activeId}},
		"expiresBefore": {Filters{ExpiresBefore: ptrTime(time.Now())}, []uuid.UUID{inactiveId}},
		"expiresAfter":  {Filters{ExpiresAfter: ptrTime(time.Now())}, []uuid.UUID{activeId}},
	} {
		t.Run("filter "+name, func(t *testing.T) {
			got, err := allEntityProvider(tm, tc.f)(db)()
			require.NoError(t, err)
			ids := make([]uuid.UUID, 0, len(got))
			for _, e := range got {
				ids = append(ids, e.Id)
			}
			assert.Equal(t, tc.want, ids)
		})
	}
}
