package commodities

import (
	"github.com/google/uuid"
)

// RestModel is a JSON API representation of the Model
type RestModel struct {
	Id                string `json:"id"`
	TemplateId        uint32 `json:"templateId"`
	MesoPrice         uint32 `json:"mesoPrice"`
	PerfectPitchPrice uint32 `json:"perfectPitchPrice"`
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

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                m.id.String(),
		TemplateId:        m.templateId,
		MesoPrice:         m.mesoPrice,
		PerfectPitchPrice: m.perfectPitchPrice,
	}, nil
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}

	builder := &ModelBuilder{}
	return builder.
		SetId(id).
		SetTemplateId(rm.TemplateId).
		SetMesoPrice(rm.MesoPrice).
		SetPerfectPitchPrice(rm.PerfectPitchPrice).
		Build(), nil
}
