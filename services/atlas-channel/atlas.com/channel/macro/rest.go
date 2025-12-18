package macro

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"strconv"
)

type RestModel struct {
	Id       uint32 `json:"-"`
	Name     string `json:"name"`
	Shout    bool   `json:"shout"`
	SkillId1 uint32 `json:"skillId1"`
	SkillId2 uint32 `json:"skillId2"`
	SkillId3 uint32 `json:"skillId3"`
}

func (r RestModel) GetName() string {
	return "macros"
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
	return RestModel{
		Id:       m.Id(),
		Name:     m.Name(),
		Shout:    m.Shout(),
		SkillId1: uint32(m.SkillId1()),
		SkillId2: uint32(m.SkillId2()),
		SkillId3: uint32(m.SkillId3()),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:       rm.Id,
		name:     rm.Name,
		shout:    rm.Shout,
		skillId1: skill.Id(rm.SkillId1),
		skillId2: skill.Id(rm.SkillId2),
		skillId3: skill.Id(rm.SkillId3),
	}, nil
}
