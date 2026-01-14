package storage

import (
	"atlas-storage/asset"
	"strconv"
)

type RestModel struct {
	Id        string                 `json:"-"`
	WorldId   byte                   `json:"world_id"`
	AccountId uint32                 `json:"account_id"`
	Capacity  uint32                 `json:"capacity"`
	Mesos     uint32                 `json:"mesos"`
	Assets    []asset.BaseRestModel  `json:"assets"`
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
