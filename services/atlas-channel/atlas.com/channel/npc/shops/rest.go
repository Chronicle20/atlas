package shops

import (
	"atlas-channel/npc/shops/commodities"
	"fmt"
)

// RestModel is a JSON API representation of the Model
type RestModel struct {
	Id          string                  `json:"id"`
	NpcId       uint32                  `json:"npcId"`
	Commodities []commodities.RestModel `json:"commodities"`
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
	return "shops"
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	commodityRest := make([]commodities.RestModel, 0)
	for _, c := range m.Commodities() {
		cr, err := commodities.Transform(c)
		if err != nil {
			return RestModel{}, err
		}
		commodityRest = append(commodityRest, cr)
	}

	return RestModel{
		Id:          fmt.Sprintf("shop-%d", m.NpcId()),
		NpcId:       m.NpcId(),
		Commodities: commodityRest,
	}, nil
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	commodityModels := make([]commodities.Model, 0)
	for _, cr := range rm.Commodities {
		cm, err := commodities.Extract(cr)
		if err != nil {
			return Model{}, err
		}
		commodityModels = append(commodityModels, cm)
	}

	return NewBuilder(rm.NpcId).
		SetCommodities(commodityModels).
		Build(), nil
}
