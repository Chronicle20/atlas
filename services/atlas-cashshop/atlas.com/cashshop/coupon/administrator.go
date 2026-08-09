package coupon

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// reserveUse claims one use of a coupon ATOMICALLY and reports whether the
// claim succeeded.
//
// The WHERE clause carries the max-uses predicate, so the check and the
// increment are one statement and RowsAffected is the verdict. A
// read-then-write here is a race: two concurrent redemptions of a
// max_uses = 1 coupon would both read 0, both write 1, and both succeed.
// This is FR-5.5 and it is explicitly banned in review.
func reserveUse(db *gorm.DB, t tenant.Model, id uuid.UUID) (bool, error) {
	res := db.Model(&Entity{}).
		Where("id = ? AND tenant_id = ? AND (max_uses IS NULL OR redemption_count < max_uses)", id, t.Id()).
		UpdateColumns(map[string]interface{}{
			"redemption_count": gorm.Expr("redemption_count + 1"),
			"updated_at":       time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// releaseUse gives a claimed use back. It is the compensation for a redemption
// that failed AFTER the reservation but is only reachable when the surrounding
// transaction did NOT roll back — inside one ExecuteTransaction the ROLLBACK
// already undoes the increment. It exists for the out-of-transaction paths and
// is guarded against underflow because redemption_count is unsigned.
func releaseUse(db *gorm.DB, t tenant.Model, id uuid.UUID) error {
	return db.Model(&Entity{}).
		Where("id = ? AND tenant_id = ? AND redemption_count > 0", id, t.Id()).
		UpdateColumns(map[string]interface{}{
			"redemption_count": gorm.Expr("redemption_count - 1"),
			"updated_at":       time.Now(),
		}).Error
}

// CreateEntity inserts one coupon. It is EXPORTED because coupon/batch calls it
// across the package boundary during bulk generation.
//
// The driver error is returned UNWRAPPED so redemption.IsUniqueViolation can
// classify a (tenant_id, code) collision as a retryable duplicate rather than
// an internal error. Do not wrap it in anything that hides the *pgconn.PgError.
func CreateEntity(db *gorm.DB, t tenant.Model, m Model) (Model, error) {
	e := &Entity{
		Id:              m.Id(),
		TenantId:        t.Id(),
		BatchId:         batchIdOrNil(m.BatchId()),
		Code:            m.Code(),
		Description:     m.Description(),
		Active:          m.Active(),
		StartsAt:        m.StartsAt(),
		ExpiresAt:       m.ExpiresAt(),
		MaxUses:         m.MaxUses(),
		RedemptionCount: m.RedemptionCount(),
		Rewards:         m.Rewards(),
	}

	// Select("*") forces EVERY column into the INSERT. GORM otherwise omits a
	// zero-valued field whose column carries a `default:` tag and substitutes
	// the default — which is how a coupon created inactive would come back
	// active. Entity.Active no longer carries such a tag; this keeps the insert
	// honest if one is ever added back.
	if err := db.Select("*").Create(e).Error; err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func batchIdOrNil(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// updateEntity applies the admin-editable fields of m to the coupon it
// identifies. It writes an EXPLICIT column set rather than saving the whole
// row, so it can never clobber redemption_count — that column belongs to
// reserveUse/releaseUse alone, and a Save() here would race them.
//
// code and batch_id are not editable: the code is the identity a player types,
// and the batch is the generation that produced the row.
func updateEntity(db *gorm.DB, t tenant.Model, m Model) (Model, error) {
	res := db.Model(&Entity{}).
		Where("id = ? AND tenant_id = ?", m.Id(), t.Id()).
		UpdateColumns(map[string]interface{}{
			"description": m.Description(),
			"active":      m.Active(),
			"starts_at":   m.StartsAt(),
			"expires_at":  m.ExpiresAt(),
			"max_uses":    m.MaxUses(),
			"rewards":     m.Rewards(),
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return Model{}, res.Error
	}
	if res.RowsAffected == 0 {
		return Model{}, gorm.ErrRecordNotFound
	}

	e, err := byIdEntityProvider(t, m.Id())(db)()
	if err != nil {
		return Model{}, err
	}
	return Make(e)
}

// ErrHasRedemptions is returned by deleteEntity when the coupon has already
// been redeemed. Deleting it would orphan the redemption rows that are the
// audit trail of those grants, so the delete is refused; Task 23 maps this to
// HTTP 409.
var ErrHasRedemptions = errors.New("coupon has redemptions")

// redemptionsTable is coupon_redemptions, referenced by NAME rather than by
// type: package coupon/redemption imports package coupon (its Entity embeds
// coupon.Rewards), so importing it back here would be an import cycle.
const redemptionsTable = "coupon_redemptions"

func deleteEntity(db *gorm.DB, t tenant.Model, id uuid.UUID) error {
	var count int64
	err := db.Table(redemptionsTable).
		Where("tenant_id = ? AND coupon_id = ?", t.Id(), id).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrHasRedemptions
	}

	res := db.Where("id = ? AND tenant_id = ?", id, t.Id()).Delete(&Entity{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
