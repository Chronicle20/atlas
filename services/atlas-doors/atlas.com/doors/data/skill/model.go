package skill

import "atlas-doors/data/skill/effect"

// Model is an immutable skill value carrying its per-level effects.
type Model struct {
	id      uint32
	effects []effect.Model
}

func (m Model) Id() uint32 {
	return m.id
}

// Effects returns the per-level effect list. Level l (1-based) maps to
// Effects()[l-1].
func (m Model) Effects() []effect.Model {
	return m.effects
}
