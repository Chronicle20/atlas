package guild

import (
	"atlas-login/guild/member"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id       uint32             `json:"-"`
	LeaderId uint32             `json:"leaderId"`
	Members  []member.RestModel `json:"members"`
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
	members, err := model.SliceMap(member.Extract)(model.FixedProvider(rm.Members))()()
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:       rm.Id,
		leaderId: rm.LeaderId,
		members:  members,
	}, nil
}
