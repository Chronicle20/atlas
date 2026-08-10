package redemption

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Create inserts one redemption row. It is EXPORTED because package coupon's
// redemption flow calls it across the package boundary.
//
// The driver error is returned UNWRAPPED so IsUniqueViolation can classify a
// (tenant_id, coupon_id, account_id) collision as the same-account race loser
// rather than an internal error. Do not wrap it in anything that hides the
// *pgconn.PgError.
func Create(db *gorm.DB, t tenant.Model, m Model) (Model, error) {
	e := &Entity{
		Id:             m.Id(),
		TenantId:       t.Id(),
		CouponId:       m.CouponId(),
		AccountId:      m.AccountId(),
		CharacterId:    m.CharacterId(),
		TransactionId:  m.TransactionId(),
		RewardsGranted: m.RewardsGranted(),
		RedeemedAt:     m.RedeemedAt(),
	}

	if err := db.Create(e).Error; err != nil {
		return Model{}, err
	}
	return Make(*e)
}

// CountByCouponAndAccount answers step 5 of the FR-5.4 ladder — "has this
// account already redeemed this coupon?" — as a friendly-error fast path. The
// real enforcement is the unique index the Create above collides with.
func CountByCouponAndAccount(db *gorm.DB, t tenant.Model, couponId uuid.UUID, accountId uint32) (int64, error) {
	var count int64
	err := db.Model(&Entity{}).
		Where("tenant_id = ? AND coupon_id = ? AND account_id = ?", t.Id(), couponId, accountId).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
