package redemption

import (
	"atlas-cashshop/coupon/reward"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	id             uuid.UUID
	couponId       uuid.UUID
	accountId      uint32
	characterId    uint32
	transactionId  uuid.UUID
	rewardsGranted reward.Rewards
	redeemedAt     time.Time
}

func NewBuilder(couponId uuid.UUID, accountId uint32, characterId uint32) *Builder {
	return &Builder{couponId: couponId, accountId: accountId, characterId: characterId}
}

func (b *Builder) SetId(id uuid.UUID) *Builder                 { b.id = id; return b }
func (b *Builder) SetTransactionId(id uuid.UUID) *Builder      { b.transactionId = id; return b }
func (b *Builder) SetRewardsGranted(r reward.Rewards) *Builder { b.rewardsGranted = r; return b }
func (b *Builder) SetRedeemedAt(t time.Time) *Builder          { b.redeemedAt = t; return b }

func (b *Builder) Build() (Model, error) {
	if b.couponId == uuid.Nil {
		return Model{}, fmt.Errorf("%w: couponId is required", ErrInvalidRedemption)
	}
	if b.accountId == 0 {
		return Model{}, fmt.Errorf("%w: accountId is required", ErrInvalidRedemption)
	}
	if len(b.rewardsGranted) == 0 {
		return Model{}, fmt.Errorf("%w: rewardsGranted must not be empty", ErrInvalidRedemption)
	}
	return Model{
		id:             b.id,
		couponId:       b.couponId,
		accountId:      b.accountId,
		characterId:    b.characterId,
		transactionId:  b.transactionId,
		rewardsGranted: b.rewardsGranted,
		redeemedAt:     b.redeemedAt,
	}, nil
}
