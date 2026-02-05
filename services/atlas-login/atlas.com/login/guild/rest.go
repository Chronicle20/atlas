package guild

import (
	"atlas-login/guild/member"
	"atlas-login/guild/title"
	"strconv"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
)

type RestModel struct {
	Id                  uint32             `json:"-"`
	WorldId             world.Id           `json:"worldId"`
	Name                string             `json:"name"`
	Notice              string             `json:"notice"`
	Points              uint32             `json:"points"`
	Capacity            uint32             `json:"capacity"`
	Logo                uint16             `json:"logo"`
	LogoColor           byte               `json:"logoColor"`
	LogoBackground      uint16             `json:"logoBackground"`
	LogoBackgroundColor byte               `json:"logoBackgroundColor"`
	LeaderId            uint32             `json:"leaderId"`
	Members             []member.RestModel `json:"members"`
	Titles              []title.RestModel  `json:"titles"`
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
