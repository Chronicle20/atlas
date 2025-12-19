package asset

import (
	"atlas-cashshop/cashshop/item"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel represents a cash shop inventory asset for REST API
type RestModel struct {
	Id            uuid.UUID      `json:"-"`
	CompartmentId uuid.UUID      `json:"compartmentId"`
	Item          item.RestModel `json:"-"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "assets"
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
			Type: "items",
			Name: "item",
		},
	}
}

// GetReferencedIDs returns the referenced IDs for this resource
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{
		{
			ID:   r.Item.GetID(),
			Type: r.Item.GetName(),
			Name: "item",
		},
	}
}

// GetReferencedStructs returns the referenced structs for this resource
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{r.Item}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	if name == "item" {
		var item item.RestModel
		if err := item.SetID(ID); err != nil {
			return err
		}
		r.Item = item
	}
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets the referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if r.Item.GetID() != "" {
		if refMap, ok := references["items"]; ok {
			if ref, ok := refMap[r.Item.GetID()]; ok {
				err := jsonapi.ProcessIncludeData(&r.Item, ref, references)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Transform converts an asset.Model to an RestModel
func Transform(a Model) (RestModel, error) {
	item, err := item.Transform(a.Item())
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:            a.Id(),
		CompartmentId: a.CompartmentId(),
		Item:          item,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	item, err := item.Extract(rm.Item)
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:            rm.Id,
		compartmentId: rm.CompartmentId,
		item:          item,
	}, nil
}
