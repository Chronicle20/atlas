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
	es, err := model.SliceMap(effect.Transform)(model.FixedProvider(m.Effects))()()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:            m.Id,
		Action:        m.Action,
		Element:       m.Element,
		AnimationTime: m.AnimationTime,
		Effects:       es,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	es, err := model.SliceMap(effect.Extract)(model.FixedProvider(rm.Effects))()()
	if err != nil {
		return Model{}, err
	}

	return Model{
		Id:            rm.Id,
		Action:        rm.Action,
		Element:       rm.Element,
		AnimationTime: rm.AnimationTime,
		Effects:       es,
	}, nil
}
