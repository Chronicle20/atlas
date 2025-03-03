package slot

import (
	"atlas-character/equipable"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Type          string               `json:"-"`
	Position      Position             `json:"position"`
	Equipable     *equipable.RestModel `json:"-"`
	CashEquipable *equipable.RestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "equipment"
}

func (r RestModel) GetID() string {
	return r.Type
}

func (r *RestModel) SetID(strType string) error {
	r.Type = strType
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "equipable",
			Name: "equipable",
		},
		{
			Type: "cashEquipable",
			Name: "cashEquipable",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	if r.Equipable != nil {
		result = append(result, jsonapi.ReferenceID{
			ID:   r.Equipable.GetID(),
			Type: "equipable",
			Name: "equipable",
		})
	}
	if r.CashEquipable != nil {
		result = append(result, jsonapi.ReferenceID{
			ID:   r.CashEquipable.GetID(),
			Type: "cashEquipable",
			Name: "cashEquipable",
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	if r.Equipable != nil {
		result = append(result, r.Equipable)
	}
	if r.CashEquipable != nil {
		result = append(result, r.CashEquipable)
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(references []jsonapi.Data) error {
	return nil
}

func Transform(model Model) (RestModel, error) {
	var rem *equipable.RestModel
	var rcem *equipable.RestModel
	if model.Equipable != nil {
		m, err := equipable.Transform(*model.Equipable)
		if err != nil {
			return RestModel{}, err
		}
		rem = &m
	}
	if model.CashEquipable != nil {
		m, err := equipable.Transform(*model.CashEquipable)
		if err != nil {
			return RestModel{}, err
		}
		rcem = &m
	}

	rm := RestModel{
		Position:      model.Position,
		Equipable:     rem,
		CashEquipable: rcem,
	}
	return rm, nil
}

func Extract(model RestModel) (Model, error) {
	m := Model{Position: model.Position}
	if model.Equipable != nil {
		e, err := equipable.Extract(*model.Equipable)
		if err != nil {
			return m, err
		}
		m.Equipable = &e
	}
	if model.CashEquipable != nil {
		e, err := equipable.Extract(*model.CashEquipable)
		if err != nil {
			return m, err
		}
		m.CashEquipable = &e
	}
	return m, nil
}
