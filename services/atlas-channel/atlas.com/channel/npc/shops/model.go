package shops

import "atlas-channel/npc/shops/commodities"

type Model struct {
	npcId       uint32
	commodities []commodities.Model
}

// NpcId returns a pointer to the model's npcId
func (m *Model) NpcId() uint32 {
	return m.npcId
}

// Commodities returns a pointer to the model's commodities
func (m *Model) Commodities() []commodities.Model {
	return m.commodities
}
