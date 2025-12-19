package inventory

import (
	"atlas-cashshop/character/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id           uuid.UUID               `json:"-"`
	CharacterId  uint32                  `json:"characterId"`
	Compartments []compartment.RestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "inventories"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "compartments",
			Name: "compartments",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Compartments {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Compartments {
		result = append(result, r.Compartments[key])
	}

	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "compartments" {
		for _, idStr := range IDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return err
			}
			r.Compartments = append(r.Compartments, compartment.RestModel{Id: id})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["compartments"]; ok {
		compartments := make([]compartment.RestModel, 0)
		for _, ri := range r.Compartments {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				compartments = append(compartments, wip)
			}
		}
		r.Compartments = compartments
	}
	return nil
}

func Transform(m Model) (RestModel, error) {
	cs := make([]compartment.RestModel, 0)
	for _, v := range m.compartments {
		c, err := compartment.Transform(v)
		if err != nil {
			return RestModel{}, nil
		}
		cs = append(cs, c)
	}

	return RestModel{
		Id:           uuid.New(),
		CharacterId:  m.characterId,
		Compartments: cs,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	cs := make(map[inventory.Type]compartment.Model)
	for _, v := range rm.Compartments {
		c, err := compartment.Extract(v)
		if err != nil {
			return Model{}, nil
		}
		cs[c.Type()] = c
	}

	return Model{
		characterId:  rm.CharacterId,
		compartments: cs,
	}, nil
}
