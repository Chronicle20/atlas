package slot

import (
	"atlas-consumables/equipable"
	"encoding/json"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
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

var References = []string{"equipable", "cashEquipable"}

func (r RestModel) GetReferences() []jsonapi.Reference {
	references := make([]jsonapi.Reference, 0)
	for _, ref := range References {
		references = append(references, jsonapi.Reference{
			Type: ref,
			Name: ref,
		})
	}
	return references
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
	if name == "equipable" {
		id, err := strconv.Atoi(ID)
		if err != nil {
			return err
		}
		r.Equipable = &equipable.RestModel{Id: uint32(id)}
		return nil
	}
	if name == "cashEquipable" {
		id, err := strconv.Atoi(ID)
		if err != nil {
			return err
		}
		r.CashEquipable = &equipable.RestModel{Id: uint32(id)}
		return nil
	}
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["equipables"]; ok {
		for _, ref := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[ref.ID]; ok {
				var erm *equipable.RestModel
				if ref.Type == "equipable" {
					erm = r.Equipable
				}
				if ref.Type == "cashEquipable" {
					erm = r.CashEquipable
				}
				err := json.Unmarshal(data.Attributes, erm)
				if err != nil {
					return err
				}
				if ref.Type == "equipable" {
					r.Equipable = erm
				}
				if ref.Type == "cashEquipable" {
					r.CashEquipable = erm
				}
			}
		}
	}
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
