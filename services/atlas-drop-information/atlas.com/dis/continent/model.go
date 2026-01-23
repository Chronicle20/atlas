package continent

import (
	"atlas-drops-information/continent/drop"
)

type Model struct {
	id    int32
	drops []drop.Model
}

func (m Model) Id() int32 {
	return m.id
}

func (m Model) Drops() []drop.Model {
	result := make([]drop.Model, len(m.drops))
	copy(result, m.drops)
	return result
}
