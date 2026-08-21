package npc

// Model is the deploy-time NPC template read from atlas-data. Imitate
// gates whether a given NPC id is eligible for the imitate pool (design
// C-1).
type Model struct {
	id      uint32
	name    string
	imitate bool
}

func (m Model) Id() uint32    { return m.id }
func (m Model) Name() string  { return m.name }
func (m Model) Imitate() bool { return m.imitate }
