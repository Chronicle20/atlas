package guild

import (
	"strconv"
)

type RestModel struct {
	Id       uint32 `json:"-"`
	LeaderId uint32 `json:"leaderId"`
}

func (r RestModel) GetName() string {
	return "guilds"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:       rm.Id,
		leaderId: rm.LeaderId,
	}, nil
}
