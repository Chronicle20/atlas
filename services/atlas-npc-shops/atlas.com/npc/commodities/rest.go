package commodities

import (
	"github.com/google/uuid"
)

// RestModel is a JSON API representation of the Model
type RestModel struct {
	Id           string  `json:"id"`
	TemplateId   uint32  `json:"templateId"`
	MesoPrice    uint32  `json:"mesoPrice"`
	DiscountRate byte    `json:"discountRate"`
	TokenTemplateId  uint32  `json:"tokenTemplateId"`
	TokenPrice   uint32  `json:"tokenPrice"`
	Period       uint32  `json:"period"`
	LevelLimit   uint32  `json:"levelLimit"`
	UnitPrice    float64 `json:"unitPrice"`
	SlotMax      uint32  `json:"slotMax"`
}

// GetID to satisfy jsonapi.MarshalIdentifier interface
func (r RestModel) GetID() string {
	return r.Id
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName to satisfy jsonapi.EntityNamer interface
func (r RestModel) GetName() string {
	return "commodities"
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:              m.Id().String(),
		TemplateId:      m.TemplateId(),
		MesoPrice:       m.MesoPrice(),
		DiscountRate:    m.DiscountRate(),
		TokenTemplateId: m.TokenTemplateId(),
		TokenPrice:      m.TokenPrice(),
		Period:          m.Period(),
		LevelLimit:      m.LevelLimit(),
		UnitPrice:       m.UnitPrice(),
		SlotMax:         m.SlotMax(),
	}, nil
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}

	return NewBuilder().
		SetId(id).
		SetTemplateId(rm.TemplateId).
		SetMesoPrice(rm.MesoPrice).
		SetDiscountRate(rm.DiscountRate).
		SetTokenTemplateId(rm.TokenTemplateId).
		SetTokenPrice(rm.TokenPrice).
		SetPeriod(rm.Period).
		SetLevelLimit(rm.LevelLimit).
		SetUnitPrice(rm.UnitPrice).
		SetSlotMax(rm.SlotMax).
		Build()
}
