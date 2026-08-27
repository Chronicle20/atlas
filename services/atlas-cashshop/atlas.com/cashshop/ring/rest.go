package ring

import (
	"time"

	"github.com/google/uuid"
)

// RestModel is the read-only view of one half of a ring pair (PRD §5.4: this
// closes the open question of which service owns the ring query surface --
// atlas-cashshop, since it already owns the write path via
// cashshop.PurchaseRingAndEmit). There is deliberately no write route: a
// ring pair is created only by the purchase transaction, and a REST route
// that could create or edit one would be a way to fabricate a pair outside
// it.
type RestModel struct {
	Id                 uuid.UUID `json:"-"`
	PairId             uuid.UUID `json:"pairId"`
	CharacterId        uint32    `json:"characterId"`
	PartnerCharacterId uint32    `json:"partnerCharacterId"`
	AssetId            uint32    `json:"assetId"`
	ItemTemplateId     uint32    `json:"itemTemplateId"`
	RingType           string    `json:"ringType"`
	State              string    `json:"state"`
	CreatedAt          time.Time `json:"createdAt"`
	CashId             int64     `json:"cashId"`
	PartnerCashId      int64     `json:"partnerCashId"`
	PartnerName        string    `json:"partnerName"`
}

func (r RestModel) GetName() string {
	return "rings"
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
		Id:                 m.Id(),
		PairId:             m.PairId(),
		CharacterId:        m.CharacterId(),
		PartnerCharacterId: m.PartnerCharacterId(),
		AssetId:            m.AssetId(),
		ItemTemplateId:     m.ItemTemplateId(),
		RingType:           string(m.Type()),
		State:              string(m.State()),
		CreatedAt:          m.CreatedAt(),
		CashId:             m.CashId(),
		PartnerCashId:      m.PartnerCashId(),
		PartnerName:        m.PartnerName(),
	}, nil
}

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	out := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
