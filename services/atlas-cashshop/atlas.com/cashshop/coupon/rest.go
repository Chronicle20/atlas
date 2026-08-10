package coupon

import (
	"atlas-cashshop/coupon/reward"
	"time"

	"github.com/google/uuid"
)

// RestModel is the admin representation of a coupon.
//
// Rewards is reward.Rewards VERBATIM — the same document the jsonb column
// stores — so an admin editing a bundle sees exactly what is persisted. There
// is deliberately no second, REST-only reward shape to keep in step.
//
// There is NO redeem representation and no redeem route anywhere in this
// package: a REST redeem would be an unauthenticated reward faucet. The packet
// path (Processor.RedeemAndEmit) is the only trigger.
type RestModel struct {
	Id              uuid.UUID      `json:"-"`
	BatchId         *uuid.UUID     `json:"batchId,omitempty"`
	Code            string         `json:"code"`
	Description     string         `json:"description,omitempty"`
	Active          bool           `json:"active"`
	StartsAt        *time.Time     `json:"startsAt,omitempty"`
	ExpiresAt       *time.Time     `json:"expiresAt,omitempty"`
	MaxUses         *uint32        `json:"maxUses,omitempty"`
	RedemptionCount uint32         `json:"redemptionCount"`
	Rewards         reward.Rewards `json:"rewards"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

func (r RestModel) GetName() string {
	return "coupons"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID tolerates an empty id because api2go calls it unconditionally while
// unmarshalling, and a POST body legitimately carries no id.
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
		Id:              m.Id(),
		BatchId:         batchIdOrNil(m.BatchId()),
		Code:            m.Code(),
		Description:     m.Description(),
		Active:          m.Active(),
		StartsAt:        m.StartsAt(),
		ExpiresAt:       m.ExpiresAt(),
		MaxUses:         m.MaxUses(),
		RedemptionCount: m.RedemptionCount(),
		Rewards:         m.Rewards(),
		CreatedAt:       m.CreatedAt(),
		UpdatedAt:       m.UpdatedAt(),
	}, nil
}

// Extract builds the domain model from an admin request body. It runs the
// Builder, so every invariant the Builder enforces (plausible code, non-empty
// valid rewards, expiresAt after startsAt, maxUses at or above the redemption
// count) rejects a bad request here rather than at the INSERT.
//
// redemptionCount is NOT taken from the body: it is owned by reserveUse. A
// PATCH handler supplies the stored count separately so the maxUses check has
// something real to compare against.
func Extract(rm RestModel) (Model, error) {
	batchId := uuid.Nil
	if rm.BatchId != nil {
		batchId = *rm.BatchId
	}
	return NewBuilder(rm.Code).
		SetId(rm.Id).
		SetBatchId(batchId).
		SetDescription(rm.Description).
		SetActive(rm.Active).
		SetStartsAt(rm.StartsAt).
		SetExpiresAt(rm.ExpiresAt).
		SetMaxUses(rm.MaxUses).
		SetRewards(rm.Rewards).
		Build()
}
