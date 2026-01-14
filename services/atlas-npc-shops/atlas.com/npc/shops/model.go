package shops

import "atlas-npc/commodities"

type Model struct {
	npcId       uint32
	commodities []commodities.Model
	recharger   bool
}

// NpcId returns the model's npcId
func (m Model) NpcId() uint32 {
	return m.npcId
}

// Commodities returns the model's commodities
func (m Model) Commodities() []commodities.Model {
	return m.commodities
}

// Recharger returns whether rechargeables can be recharged at this shop
func (m Model) Recharger() bool {
	return m.recharger
}
