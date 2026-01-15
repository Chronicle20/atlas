package storage

import (
	"atlas-storage/asset"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id        string                `json:"-"`
	WorldId   byte                  `json:"world_id"`
	AccountId uint32                `json:"account_id"`
	Capacity  uint32                `json:"capacity"`
	Mesos     uint32                `json:"mesos"`
	Assets    []asset.BaseRestModel `json:"-"`
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
	if refMap, ok := references["storage_assets"]; ok {
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

// Transform converts a Model to a RestModel
// Assets must already be decorated with reference data before calling this
func Transform(m Model) (RestModel, error) {
	baseRestAssets, err := asset.TransformAllToBaseRestModel(m.Assets())
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:        m.Id().String(),
		WorldId:   m.WorldId(),
		AccountId: m.AccountId(),
		Capacity:  m.Capacity(),
		Mesos:     m.Mesos(),
		Assets:    baseRestAssets,
	}, nil
}

// InputRestModel for creating/updating storage
type InputRestModel struct {
	WorldId   byte   `json:"world_id"`
	AccountId uint32 `json:"account_id"`
	Capacity  uint32 `json:"capacity,omitempty"`
	Mesos     uint32 `json:"mesos,omitempty"`
}

func (r InputRestModel) GetName() string {
	return "storages"
}

func (r InputRestModel) GetID() string {
	return strconv.Itoa(int(r.AccountId))
}

func (r *InputRestModel) SetID(id string) error {
	accountId, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.AccountId = uint32(accountId)
	return nil
}
