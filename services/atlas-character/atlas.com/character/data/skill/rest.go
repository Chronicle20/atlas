package skill

import (
	"atlas-character/data/skill/effect"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func Extract(rm RestModel) (Model, error) {
	es, err := model.SliceMap(effect.Extract)(model.FixedProvider(rm.Effects))()()
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:            rm.Id,
		action:        rm.Action,
		element:       rm.Element,
		animationTime: rm.AnimationTime,
		effects:       es,
	}, nil
}

// Transform is the inverse of Extract: it converts the domain Model back into
// the RestModel representation.
func Transform(m Model) (RestModel, error) {
	es, err := model.SliceMap(effect.Transform)(model.FixedProvider(m.effects))()()
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
