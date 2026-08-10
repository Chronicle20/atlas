package redemption

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor is the READ side of the redemption audit trail. It deliberately
// exposes no way to create, edit or delete a row: redemptions are written only
// by the packet-driven redemption transaction in package coupon, and a REST
// route that could write one would be a way to fabricate an audit record.
type Processor interface {
	ByCouponId(couponId uuid.UUID, page model.Page) (model.Paged[Model], error)
	ByAccountId(accountId uint32, page model.Page) (model.Paged[Model], error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: tenant.MustFromContext(ctx)}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByCouponId(couponId uuid.UUID, page model.Page) (model.Paged[Model], error) {
	ep := byCouponIdPagedProvider(p.t, couponId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())()
}

func (p *ProcessorImpl) ByAccountId(accountId uint32, page model.Page) (model.Paged[Model], error) {
	ep := byAccountIdPagedProvider(p.t, accountId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())()
}

// CountByBatchId counts the redemptions of every coupon belonging to one
// batch. It is EXPORTED because package coupon/batch reports the number on its
// resource and the coupon_redemptions table lives here.
//
// It counts redemption ROWS rather than summing coupons.redemption_count:
// releaseUse can decrement that column without deleting a row, so the sum is
// an approximation of the audit trail while a COUNT over the rows IS the audit
// trail.
func CountByBatchId(db *gorm.DB, t tenant.Model, batchId uuid.UUID) (int64, error) {
	var count int64
	err := db.Model(&Entity{}).
		Joins("JOIN coupons ON coupons.id = coupon_redemptions.coupon_id AND coupons.tenant_id = coupon_redemptions.tenant_id").
		Where("coupon_redemptions.tenant_id = ? AND coupons.batch_id = ?", t.Id(), batchId).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
