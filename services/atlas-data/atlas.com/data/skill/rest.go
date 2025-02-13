package skill

import (
	"atlas-data/skill/effect"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id            uint32             `json:"-"`
	Action        bool               `json:"action"`
	Element       string             `json:"element"`
	AnimationTime uint32             `json:"animationTime"`
	Effects       []effect.RestModel `json:"effects"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	es, err := model.SliceMap(effect.Transform)(model.FixedProvider(m.Effects()))()()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:            m.id,
		Action:        m.action,
		Element:       m.element,
		AnimationTime: m.animationTime,
		Effects:       es,
	}, nil
}
