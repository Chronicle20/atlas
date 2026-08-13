package redemption

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// byCouponIdPagedProvider pages every redemption of one coupon, newest last,
// for GET /coupons/{id}/redemptions.
//
// The ORDER BY carries an id tiebreaker: redeemed_at is not unique (two
// accounts can redeem inside the same clock tick) and this ordering is paged
// on, so without it a page boundary could drop or duplicate a row.
func byCouponIdPagedProvider(t tenant.Model, couponId uuid.UUID, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](
			db.Where("tenant_id = ? AND coupon_id = ?", t.Id(), couponId).Order("redeemed_at, id"), page)
	}
}

// byAccountIdPagedProvider pages every redemption an account has made, for
// GET /coupon-redemptions?filter[accountId]=.
func byAccountIdPagedProvider(t tenant.Model, accountId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](
			db.Where("tenant_id = ? AND account_id = ?", t.Id(), accountId).Order("redeemed_at, id"), page)
	}
}

// A per-coupon redemption COUNT deliberately has no provider here. The count is
// materialized on the coupon row itself (coupon.Model.redemptionCount, owned by
// reserveUse's atomic increment) and served straight from it by
// coupon.RestModel.RedemptionCount. Counting the redemption table instead would
// be a second, racier source of the same number.
