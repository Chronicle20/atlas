package redemption

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// byCouponIdProvider lists every redemption of one coupon, newest last, for
// GET /coupons/{id}/redemptions.
func byCouponIdProvider(t tenant.Model, couponId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("tenant_id = ? AND coupon_id = ?", t.Id(), couponId).
			Order("redeemed_at").
			Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// byAccountIdProvider lists every redemption an account has made, for
// GET /coupon-redemptions?filter[accountId]=.
func byAccountIdProvider(t tenant.Model, accountId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("tenant_id = ? AND account_id = ?", t.Id(), accountId).
			Order("redeemed_at").
			Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// countByCouponIdProvider counts a coupon's redemptions without loading them —
// the admin list shows the number, not the rows.
func countByCouponIdProvider(t tenant.Model, couponId uuid.UUID) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		var count int64
		err := db.Model(&Entity{}).
			Where("tenant_id = ? AND coupon_id = ?", t.Id(), couponId).
			Count(&count).Error
		if err != nil {
			return model.ErrorProvider[int64](err)
		}
		return model.FixedProvider(count)
	}
}
