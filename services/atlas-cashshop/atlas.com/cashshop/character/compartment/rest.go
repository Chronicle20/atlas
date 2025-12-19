package compartment

import (
	"atlas-cashshop/asset"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id            uuid.UUID             `json:"-"`
	InventoryType inventory.Type        `json:"type"`
	Capacity      uint32                `json:"capacity"`
	Assets        []asset.BaseRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "compartments"
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
			Type: "assets",
			Name: "assets",
		},
	}
}

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

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}

	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, asset.BaseRestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]asset.BaseRestModel, 0)
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

func Transform(m Model) (RestModel, error) {
	as, err := model.SliceMap(asset.Transform)(model.FixedProvider(m.assets))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:            m.id,
		InventoryType: m.inventoryType,
		Capacity:      m.capacity,
		Assets:        as,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	as, err := model.SliceMap(asset.Extract)(model.FixedProvider(rm.Assets))(model.ParallelMap())()
	if err != nil {
		return Model{}, nil
	}

	return Model{
		id:            rm.Id,
		inventoryType: rm.InventoryType,
		capacity:      rm.Capacity,
		assets:        as,
	}, nil
}
