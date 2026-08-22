package compartment

import (
	"atlas-channel/asset"
	"strconv"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
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

// accommodationInputRestModel is the POST body for
// characters/{characterId}/inventory/accommodation — one item per request,
// mirroring atlas-inventory's AccommodationInputRestModel.
type accommodationInputRestModel struct {
	Id    string                       `json:"-"`
	Items []accommodationItemRestModel `json:"items"`
}

type accommodationItemRestModel struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

func (accommodationInputRestModel) GetName() string                          { return "inventoryAccommodations" }
func (r accommodationInputRestModel) GetID() string                          { return r.Id }
func (r *accommodationInputRestModel) SetID(id string) error                 { r.Id = id; return nil }
func (r *accommodationInputRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (r *accommodationInputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// accommodationOutputRestModel reports the overall verdict atlas-inventory
// computed for the requested item(s). CanAccommodate only reads Accommodated.
type accommodationOutputRestModel struct {
	Id           string                         `json:"-"`
	Accommodated bool                           `json:"accommodated"`
	Results      []accommodationResultRestModel `json:"results"`
}

type accommodationResultRestModel struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Accommodated bool   `json:"accommodated"`
}

func (accommodationOutputRestModel) GetName() string                          { return "inventoryAccommodations" }
func (r accommodationOutputRestModel) GetID() string                          { return r.Id }
func (r *accommodationOutputRestModel) SetID(id string) error                 { r.Id = id; return nil }
func (r *accommodationOutputRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (r *accommodationOutputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
