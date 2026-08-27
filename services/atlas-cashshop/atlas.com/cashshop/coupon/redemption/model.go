package redemption

import (
	"atlas-cashshop/coupon/reward"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id             uuid.UUID
	couponId       uuid.UUID
	accountId      uint32
	characterId    uint32
	transactionId  uuid.UUID
	rewardsGranted reward.Rewards
	redeemedAt     time.Time
}

func (m Model) Id() uuid.UUID                  { return m.id }
func (m Model) CouponId() uuid.UUID            { return m.couponId }
func (m Model) AccountId() uint32              { return m.accountId }
func (m Model) CharacterId() uint32            { return m.characterId }
func (m Model) TransactionId() uuid.UUID       { return m.transactionId }
func (m Model) RewardsGranted() reward.Rewards { return m.rewardsGranted }
func (m Model) RedeemedAt() time.Time          { return m.redeemedAt }

var ErrInvalidRedemption = errors.New("invalid redemption")
