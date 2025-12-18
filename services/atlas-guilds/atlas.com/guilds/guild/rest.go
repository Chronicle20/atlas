package guild

import (
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id                  uint32             `json:"-"`
	WorldId             byte               `json:"worldId"`
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

func Transform(m Model) (RestModel, error) {
	members, err := model.SliceMap(member.Transform)(model.FixedProvider(m.Members()))()()
	if err != nil {
		return RestModel{}, err
	}
	titles, err := model.SliceMap(title.Transform)(model.FixedProvider(m.Titles()))()()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:                  m.id,
		WorldId:             m.worldId,
		Name:                m.name,
		Notice:              m.notice,
		Points:              m.points,
		Capacity:            m.capacity,
		Logo:                m.logo,
		LogoColor:           m.logoColor,
		LogoBackground:      m.logoBackground,
		LogoBackgroundColor: m.logoBackgroundColor,
		LeaderId:            m.leaderId,
		Members:             members,
		Titles:              titles,
	}, nil
}
