package mapdata

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RestModel mirrors the subset of atlas-data's map wire format the mini-game
// service needs (fieldLimit).
type RestModel struct {
	Id         _map.Id `json:"-"`
	FieldLimit uint32  `json:"fieldLimit"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = _map.Id(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:         rm.Id,
		fieldLimit: rm.FieldLimit,
	}, nil
}
