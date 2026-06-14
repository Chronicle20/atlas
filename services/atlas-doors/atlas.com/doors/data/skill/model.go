package skill

import (
	"atlas-doors/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Model is an immutable skill value carrying its per-level effects.
type Model struct {
	id      skill.Id
	effects []effect.Model
}

func (m Model) Id() skill.Id {
	return m.id
}

// Effects returns the per-level effect list. Level l (1-based) maps to
// Effects()[l-1].
func (m Model) Effects() []effect.Model {
	return m.effects
}
