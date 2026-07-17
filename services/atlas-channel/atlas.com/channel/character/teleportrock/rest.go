package teleportrock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type RestModel struct {
	Id      string    `json:"-"`
	Regular []_map.Id `json:"regular"`
	Vip     []_map.Id `json:"vip"`
}

func (r RestModel) GetName() string {
	return "teleport-rock-maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return NewModel(rm.Regular, rm.Vip), nil
}
