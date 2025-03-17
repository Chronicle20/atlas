package skill

import "atlas-data/skill/effect"

type Model struct {
	Id            uint32         `json:"id"`
	Action        bool           `json:"action"`
	Element       string         `json:"element"`
	AnimationTime uint32         `json:"animation_time"`
	Effects       []effect.Model `json:"effects"`
}

func (m Model) GetId() uint32 {
	return m.Id
}
