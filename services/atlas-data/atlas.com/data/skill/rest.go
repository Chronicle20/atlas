package skill

import (
	"atlas-data/skill/effect"
	"strconv"
)

type RestModel struct {
	Id            uint32             `json:"-"`
	Name          string             `json:"name"`
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
