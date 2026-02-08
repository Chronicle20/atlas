package storage

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id        string           `json:"-"`
	WorldId   world.Id         `json:"world_id"`
	AccountId uint32           `json:"account_id"`
	Capacity  uint32           `json:"capacity"`
	Mesos     uint32           `json:"mesos"`
	Assets    []AssetRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "storages"
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
			Type: "storage_assets",
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
			Name: "assets",
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
			r.Assets = append(r.Assets, AssetRestModel{Id: idStr})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["storage_assets"]; ok {
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
	Slot       int16     `json:"slot"`
	TemplateId uint32    `json:"templateId"`
	Expiration time.Time `json:"expiration"`
}

func (r AssetRestModel) GetName() string {
	return "storage_assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *AssetRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *AssetRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// GetAssetId parses the ID string to uint32
func (r AssetRestModel) GetAssetId() uint32 {
	id, _ := strconv.ParseUint(r.Id, 10, 32)
	return uint32(id)
}
