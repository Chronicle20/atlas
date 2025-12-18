package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel represents a cash shop inventory compartment for REST API
type RestModel struct {
	Id        uuid.UUID         `json:"-"`
	AccountId uint32            `json:"accountId"`
	Type      CompartmentType   `json:"type"`
	Capacity  uint32            `json:"capacity"`
	Assets    []asset.RestModel `json:"-"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "compartments"
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
			Type: "assets",
			Name: "assets",
		},
	}
}

// GetReferencedIDs returns the referenced IDs for this resource
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Assets {
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
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, asset.RestModel{Id: id})
		}
	}
	return nil
}

// SetReferencedStructs sets the referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]asset.RestModel, 0)
		for _, ri := range r.Assets {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				assets = append(assets, wip)
			}
		}
		r.Assets = assets
	}
	return nil
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	assets, err := model.SliceMap(asset.Transform)(model.FixedProvider(m.Assets()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:        m.Id(),
		AccountId: m.AccountId(),
		Type:      m.Type(),
		Capacity:  m.Capacity(),
		Assets:    assets,
	}, nil
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	assets, err := model.SliceMap(asset.Extract)(model.FixedProvider(rm.Assets))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return NewBuilder(rm.Id, rm.AccountId, rm.Type, rm.Capacity).
		SetAssets(assets).
		Build(), nil
}
