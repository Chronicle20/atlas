package shops

import "atlas-npc/commodities"

// JSONModel is the JSON representation for loading shop seed data
type JSONModel struct {
	NpcId       uint32                  `json:"npcId"`
	Recharger   bool                    `json:"recharger"`
	Commodities []commodities.JSONModel `json:"commodities"`
}
