package inventory

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// AssetRestModel mirrors the subset of atlas-inventory's "assets" resource
// a deploy snapshot needs — slot/templateId, matching the narrower shape
// used by services/atlas-channel/atlas.com/channel/asset (which has no
// relationships block of its own, so no SetToOneReferenceID /
// SetToManyReferenceIDs stubs are required here).
type AssetRestModel struct {
	Id         uint32 `json:"-"`
	Slot       int16  `json:"slot"`
	TemplateId uint32 `json:"templateId"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// CompartmentRestModel mirrors atlas-inventory's "compartments" resource:
// a type plus its assets relationship (see
// services/atlas-channel/atlas.com/channel/compartment/rest.go).
type CompartmentRestModel struct {
	Id            uuid.UUID        `json:"-"`
	InventoryType inventory.Type   `json:"type"`
	Assets        []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id.String()
}

func (r *CompartmentRestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func (r CompartmentRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "assets",
			Name: "assets",
		},
	}
}

func (r CompartmentRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
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

func (r CompartmentRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

func (r *CompartmentRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *CompartmentRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	refMap, ok := references["assets"]
	if !ok {
		return nil
	}
	assets := make([]AssetRestModel, 0, len(r.Assets))
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
	return nil
}

// RestModel mirrors atlas-inventory's "inventories" resource: the
// character's compartments relationship (design §6.1 correction).
type RestModel struct {
	Id           uuid.UUID              `json:"-"`
	CharacterId  uint32                 `json:"characterId"`
	Compartments []CompartmentRestModel `json:"-"`
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

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "compartments" {
		for _, idStr := range IDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return err
			}
			r.Compartments = append(r.Compartments, CompartmentRestModel{Id: id})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	refMap, ok := references["compartments"]
	if !ok {
		return nil
	}
	compartments := make([]CompartmentRestModel, 0, len(r.Compartments))
	for _, ri := range r.Compartments {
		if ref, ok := refMap[ri.GetID()]; ok {
			wip := ri
			if err := jsonapi.ProcessIncludeData(&wip, ref, references); err != nil {
				return err
			}
			compartments = append(compartments, wip)
		}
	}
	r.Compartments = compartments
	return nil
}

func Extract(rm RestModel) (Model, error) {
	compartments := make(map[inventory.Type][]Asset, len(rm.Compartments))
	for _, c := range rm.Compartments {
		assets := make([]Asset, 0, len(c.Assets))
		for _, a := range c.Assets {
			assets = append(assets, Asset{slot: a.Slot, templateId: a.TemplateId})
		}
		compartments[c.InventoryType] = assets
	}
	return Model{
		characterId:  rm.CharacterId,
		compartments: compartments,
	}, nil
}
