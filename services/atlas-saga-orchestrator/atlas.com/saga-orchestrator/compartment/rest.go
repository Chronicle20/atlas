package compartment

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

// AssetRestModel represents an asset from the character inventory service
type AssetRestModel struct {
	Id            string          `json:"-"`
	Slot          int16           `json:"slot"`
	TemplateId    uint32          `json:"templateId"`
	Expiration    time.Time       `json:"expiration"`
	ReferenceId   uint32          `json:"referenceId"`
	ReferenceType string          `json:"referenceType"`
	ReferenceData json.RawMessage `json:"referenceData"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = strconv.Itoa(id)
	return nil
}

// CompartmentRestModel represents a compartment from the character inventory service
type CompartmentRestModel struct {
	Id            string           `json:"-"`
	InventoryType byte             `json:"type"`
	Capacity      uint32           `json:"capacity"`
	Assets        []AssetRestModel `json:"-"`
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
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: strconv.Itoa(id)})
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
