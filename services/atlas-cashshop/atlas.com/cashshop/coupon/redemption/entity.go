package redemption

import (
	"atlas-cashshop/coupon"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the coupon_redemptions table — one row per SUCCESSFUL redemption.
//
// The uniqueIndex on (tenant_id, coupon_id, account_id) is the DATABASE-LEVEL
// one-time-per-account rule, not a convenience: it is what makes two concurrent
// redemptions by the same account resolve to exactly one success and one
// COUPON_ALREADY_USED. The ladder check in coupon.Model is only a friendly-error
// fast path.
//
// RewardsGranted is a SNAPSHOT, not a reference, so later edits to the coupon's
// bundle never rewrite history.
type Entity struct {
	Id             uuid.UUID      `gorm:"primaryKey;type:uuid"`
	TenantId       uuid.UUID      `gorm:"not null;uniqueIndex:idx_redemptions_tenant_coupon_account,priority:1;index:idx_redemptions_tenant_account,priority:1"`
	CouponId       uuid.UUID      `gorm:"not null;type:uuid;index;uniqueIndex:idx_redemptions_tenant_coupon_account,priority:2"`
	AccountId      uint32         `gorm:"not null;uniqueIndex:idx_redemptions_tenant_coupon_account,priority:3;index:idx_redemptions_tenant_account,priority:2"`
	CharacterId    uint32         `gorm:"not null"`
	TransactionId  uuid.UUID      `gorm:"not null;type:uuid"`
	RewardsGranted coupon.Rewards `gorm:"not null;type:jsonb"`
	RedeemedAt     time.Time      `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "coupon_redemptions"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

// IsUniqueViolation reports whether err is a Postgres unique_violation (23505).
// The redemption insert treats that specific code as "this account already
// redeemed this coupon" — the race loser — and every other error as internal.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func Make(e Entity) (Model, error) {
	return Model{
		id:             e.Id,
		couponId:       e.CouponId,
		accountId:      e.AccountId,
		characterId:    e.CharacterId,
		transactionId:  e.TransactionId,
		rewardsGranted: e.RewardsGranted,
		redeemedAt:     e.RedeemedAt,
	}, nil
}
