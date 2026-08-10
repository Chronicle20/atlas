package inventory

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// AssetRestModel mirrors the subset of atlas-inventory's asset wire format
// staging needs (slot, templateId, quantity, flag). The remaining ~30 equipment
// statistic fields are intentionally absent — an unmapped json field is simply
// discarded by the decoder.
type AssetRestModel struct {
	Id         asset.Id       `json:"-"`
	Slot       slot.Position  `json:"slot"`
	TemplateId item.Id        `json:"templateId"`
	Quantity   asset.Quantity `json:"quantity"`
	Flag       uint16         `json:"flag"`
}

func (r AssetRestModel) GetName() string { return "assets" }

func (r AssetRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.Id = asset.Id(id)
	return nil
}

func (r *AssetRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *AssetRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// RestModel mirrors atlas-inventory's compartment resource. The compartment
// document ALWAYS carries an `assets` relationship, so the to-many hooks below
// are load-bearing rather than defensive: without SetToManyReferenceIDs the
// api2go decode fails outright, and without SetReferencedStructs the assets
// decode to bare ids with every attribute zeroed (task-037 failure class, see
// libs/atlas-rest/CLAUDE.md).
type RestModel struct {
	Id            uuid.UUID        `json:"-"`
	InventoryType inventory.Type   `json:"type"`
	Capacity      uint32           `json:"capacity"`
	Assets        []AssetRestModel `json:"-"`
}

func (r RestModel) GetName() string { return "compartments" }

func (r RestModel) GetID() string { return r.Id.String() }

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{{Type: "assets", Name: "assets"}}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	result := make([]jsonapi.ReferenceID, 0, len(r.Assets))
	for _, v := range r.Assets {
		result = append(result, jsonapi.ReferenceID{ID: v.GetID(), Type: v.GetName(), Name: v.GetName()})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := make([]jsonapi.MarshalIdentifier, 0, len(r.Assets))
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *RestModel) SetToManyReferenceIDs(name string, ids []string) error {
	if name != "assets" {
		return nil
	}
	for _, idStr := range ids {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return err
		}
		r.Assets = append(r.Assets, AssetRestModel{Id: asset.Id(id)})
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	refMap, ok := references["assets"]
	if !ok {
		return nil
	}
	assets := make([]AssetRestModel, 0, len(r.Assets))
	for _, ri := range r.Assets {
		ref, found := refMap[ri.GetID()]
		if !found {
			continue
		}
		wip := ri
		if err := jsonapi.ProcessIncludeData(&wip, ref, references); err != nil {
			return err
		}
		assets = append(assets, wip)
	}
	r.Assets = assets
	return nil
}

// ExtractAsset folds one wire asset into its domain view.
func ExtractAsset(rm AssetRestModel) (Asset, error) {
	return NewAsset(rm.Id, rm.Slot, rm.TemplateId, rm.Quantity, rm.Flag), nil
}

// Extract folds the wire compartment into its domain view.
func Extract(rm RestModel) (Model, error) {
	assets := make([]Asset, 0, len(rm.Assets))
	for _, a := range rm.Assets {
		ea, err := ExtractAsset(a)
		if err != nil {
			return Model{}, err
		}
		assets = append(assets, ea)
	}
	return Model{
		id:            rm.Id,
		inventoryType: rm.InventoryType,
		capacity:      rm.Capacity,
		assets:        assets,
	}, nil
}
