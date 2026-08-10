package batch

import (
	"atlas-cashshop/coupon/reward"
	"time"

	"github.com/google/uuid"
)

// RestModel is one bulk generation.
//
// Count/Prefix/Length/StartsAt/ExpiresAt/Rewards are INPUT-ONLY — they describe
// the generation request and are not stored on the batch row, so they are
// omitempty and absent from a GET. Codes is OUTPUT-ONLY and appears only on the
// POST response: it is the one moment the plaintext codes are handed back, and
// re-serving them from a later GET would turn any list request into a code
// dump.
type RestModel struct {
	Id             uuid.UUID      `json:"-"`
	Description    string         `json:"description,omitempty"`
	RequestedCount uint32         `json:"requestedCount"`
	GeneratedCount uint32         `json:"generatedCount"`
	RedeemedCount  uint32         `json:"redeemedCount"`
	CreatedAt      time.Time      `json:"createdAt"`
	Count          uint32         `json:"count,omitempty"`
	Prefix         string         `json:"prefix,omitempty"`
	Length         int            `json:"length,omitempty"`
	StartsAt       *time.Time     `json:"startsAt,omitempty"`
	ExpiresAt      *time.Time     `json:"expiresAt,omitempty"`
	Rewards        reward.Rewards `json:"rewards,omitempty"`
	Codes          []string       `json:"codes,omitempty"`
}

func (r RestModel) GetName() string {
	return "coupon-batches"
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
		Id:             m.Id(),
		Description:    m.Description(),
		RequestedCount: m.RequestedCount(),
		GeneratedCount: m.GeneratedCount(),
		RedeemedCount:  m.RedeemedCount(),
		CreatedAt:      m.CreatedAt(),
	}, nil
}

// ExtractInput reads the generation request off a POST body. It carries no
// tenant id — the tenant comes from the request headers.
func ExtractInput(rm RestModel) GenerateInput {
	return GenerateInput{
		Count:       rm.Count,
		Prefix:      rm.Prefix,
		Length:      rm.Length,
		Description: rm.Description,
		StartsAt:    rm.StartsAt,
		ExpiresAt:   rm.ExpiresAt,
		Rewards:     rm.Rewards,
	}
}
