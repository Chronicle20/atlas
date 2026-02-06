package skill

import (
	"atlas-channel/data/skill/effect"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
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
