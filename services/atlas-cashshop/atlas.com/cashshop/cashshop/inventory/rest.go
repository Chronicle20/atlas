package inventory

import (
	"atlas-cashshop/cashshop/inventory/compartment"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel represents a cash shop inventory for REST API
type RestModel struct {
	Id           uuid.UUID               `json:"-"`
	AccountId    uint32                  `json:"accountId"`
	Compartments []compartment.RestModel `json:"-"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "cash-inventories"
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID sets the resource ID
func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

// GetReferences returns the references for this resource
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "compartments",
			Name: "compartments",
		},
	}
}

// GetReferencedIDs returns the referenced IDs for this resource
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

// GetReferencedStructs returns the referenced structs for this resource
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Compartments {
		result = append(result, r.Compartments[key])
	}
	return result
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
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

// SetReferencedStructs sets the referenced structs
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

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	cs := make([]compartment.RestModel, 0)
	for _, v := range m.Compartments() {
		c, err := compartment.Transform(v)
		if err != nil {
			return RestModel{}, err
		}
		cs = append(cs, c)
	}

	return RestModel{
		Id:           uuid.New(),
		AccountId:    m.AccountId(),
		Compartments: cs,
	}, nil
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	cs := make(map[compartment.CompartmentType]compartment.Model)
	for _, v := range rm.Compartments {
		c, err := compartment.Extract(v)
		if err != nil {
			return Model{}, err
		}
		cs[c.Type()] = c
	}

	return Model{
		accountId:    rm.AccountId,
		compartments: cs,
	}, nil
}