package redemption

import (
	"atlas-cashshop/coupon/reward"
	"time"

	"github.com/google/uuid"
)

// RestModel is the audit view of one successful redemption. It is READ-ONLY:
// no route creates, edits or deletes a redemption, because a redemption row is
// the record that a grant happened and rewriting it would rewrite history.
//
// RewardsGranted is the snapshot taken at redemption time, not the coupon's
// current bundle, so a later edit to the coupon never changes what this row
// says was handed out.
type RestModel struct {
	Id             uuid.UUID      `json:"-"`
	CouponId       uuid.UUID      `json:"couponId"`
	AccountId      uint32         `json:"accountId"`
	CharacterId    uint32         `json:"characterId"`
	TransactionId  uuid.UUID      `json:"transactionId"`
	RewardsGranted reward.Rewards `json:"rewardsGranted"`
	RedeemedAt     time.Time      `json:"redeemedAt"`
}

func (r RestModel) GetName() string {
	return "coupon-redemptions"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = uuid.Nil
		return nil
	}
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id(),
		CouponId:       m.CouponId(),
		AccountId:      m.AccountId(),
		CharacterId:    m.CharacterId(),
		TransactionId:  m.TransactionId(),
		RewardsGranted: m.RewardsGranted(),
		RedeemedAt:     m.RedeemedAt(),
	}, nil
}
