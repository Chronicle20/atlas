package skill

import (
	"atlas-summons/data/skill/effect"
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

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go's
// unmarshal path when the upstream resource carries a relationships block (see
// libs/atlas-rest gotchas).
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Transform converts a Model into a RestModel. It is the inverse of Extract.
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
