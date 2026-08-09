package redemption

import (
	"atlas-cashshop/coupon"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id             uuid.UUID
	couponId       uuid.UUID
	accountId      uint32
	characterId    uint32
	transactionId  uuid.UUID
	rewardsGranted coupon.Rewards
	redeemedAt     time.Time
}

func (m Model) Id() uuid.UUID                  { return m.id }
func (m Model) CouponId() uuid.UUID            { return m.couponId }
func (m Model) AccountId() uint32              { return m.accountId }
func (m Model) CharacterId() uint32            { return m.characterId }
func (m Model) TransactionId() uuid.UUID       { return m.transactionId }
func (m Model) RewardsGranted() coupon.Rewards { return m.rewardsGranted }
func (m Model) RedeemedAt() time.Time          { return m.redeemedAt }

type Builder struct {
	id             uuid.UUID
	couponId       uuid.UUID
	accountId      uint32
	characterId    uint32
	transactionId  uuid.UUID
	rewardsGranted coupon.Rewards
	redeemedAt     time.Time
}

func NewBuilder(couponId uuid.UUID, accountId uint32, characterId uint32) *Builder {
	return &Builder{couponId: couponId, accountId: accountId, characterId: characterId}
}

func (b *Builder) SetId(id uuid.UUID) *Builder                 { b.id = id; return b }
func (b *Builder) SetTransactionId(id uuid.UUID) *Builder      { b.transactionId = id; return b }
func (b *Builder) SetRewardsGranted(r coupon.Rewards) *Builder { b.rewardsGranted = r; return b }
func (b *Builder) SetRedeemedAt(t time.Time) *Builder          { b.redeemedAt = t; return b }

var ErrInvalidRedemption = errors.New("invalid redemption")

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
