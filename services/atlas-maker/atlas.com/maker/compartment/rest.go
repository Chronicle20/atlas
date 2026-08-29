package compartment

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// assetRestModel is the subset of atlas-inventory's asset resource this
// service consumes.
type assetRestModel struct {
	Id         uint32 `json:"-"`
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`
}

func (r assetRestModel) GetName() string { return "assets" }
func (r assetRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }
func (r *assetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// RestModel mirrors atlas-inventory's compartment resource, keeping only the
// attributes and the to-many "assets" relationship this service consumes.
type RestModel struct {
	Id            uuid.UUID        `json:"-"`
	InventoryType inventory.Type   `json:"type"`
	Capacity      uint32           `json:"capacity"`
	Assets        []assetRestModel `json:"-"`
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
			r.Assets = append(r.Assets, assetRestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]assetRestModel, 0)
		for _, ri := range r.Assets {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				if err := jsonapi.ProcessIncludeData(&wip, ref, references); err != nil {
					return err
				}
				assets = append(assets, wip)
			}
		}
		r.Assets = assets
	}
	return nil
}

func Extract(rm RestModel) (Model, error) {
	as, err := model.SliceMap(extractAsset)(model.FixedProvider(rm.Assets))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:            rm.Id,
		inventoryType: rm.InventoryType,
		capacity:      rm.Capacity,
		assets:        as,
	}, nil
}

func extractAsset(rm assetRestModel) (AssetModel, error) {
	return AssetModel{templateId: item.Id(rm.TemplateId), quantity: rm.Quantity}, nil
}

// accommodationInputRestModel is the POST body for
// characters/{characterId}/inventory/accommodation, mirroring
// atlas-inventory's AccommodationInputRestModel.
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
// computed for the requested item(s), plus a per-item breakdown.
// CanAccommodate reads Accommodated; Results is decoded and exercised by
// this package's tests so a wire-shape drift with atlas-inventory fails
// loudly here.
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
