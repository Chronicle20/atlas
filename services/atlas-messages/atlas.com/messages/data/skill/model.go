package skill

import (
	"atlas-messages/data/skill/effect"
)

type Model struct {
	id            uint32
	name          string
	action        bool
	element       string
	animationTime uint32
	effects       []effect.Model
}

func (m Model) Effects() []effect.Model {
	return m.effects
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Action() bool {
	return m.action
}

func (m Model) Element() string {
	return m.element
}

func (m Model) AnimationTime() uint32 {
	return m.animationTime
}
