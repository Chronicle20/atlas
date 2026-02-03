package inventory

import (
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	CompartmentTypeEquip uint8 = 1
	CompartmentTypeUse   uint8 = 2
	CompartmentTypeSetup uint8 = 3
	CompartmentTypeETC   uint8 = 4
	CompartmentTypeCash  uint8 = 5
)

type RestModel struct {
	Id           string                 `json:"-"`
	CharacterId  uint32                 `json:"characterId"`
	Compartments []CompartmentRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "inventories"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
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
			r.Compartments = append(r.Compartments, CompartmentRestModel{Id: idStr})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["compartments"]; ok {
		compartments := make([]CompartmentRestModel, 0)
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

type CompartmentRestModel struct {
	Id       string           `json:"-"`
	Type     uint8            `json:"type"`
	Capacity uint32           `json:"capacity"`
	Assets   []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id
}

func (r *CompartmentRestModel) SetID(id string) error {
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

func (r *CompartmentRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			r.Assets = append(r.Assets, AssetRestModel{Id: idStr})
		}
	}
	return nil
}

func (r *CompartmentRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]AssetRestModel, 0)
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

type AssetRestModel struct {
	Id         string    `json:"-"`
	TemplateId uint32    `json:"templateId"`
	Slot       int16     `json:"slot"`
	Expiration time.Time `json:"expiration"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *AssetRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *AssetRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}
