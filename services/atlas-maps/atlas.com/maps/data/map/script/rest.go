package script

import (
	"strconv"

	_map "github.com/Chronicle20/atlas-constants/map"
)

type RestModel struct {
	Id               _map.Id `json:"-"`
	OnFirstUserEnter string  `json:"onFirstUserEnter"`
	OnUserEnter      string  `json:"onUserEnter"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = _map.Id(id)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		onFirstUserEnter: rm.OnFirstUserEnter,
		onUserEnter:      rm.OnUserEnter,
	}, nil
}
