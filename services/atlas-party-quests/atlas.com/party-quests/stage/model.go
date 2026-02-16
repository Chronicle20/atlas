package stage

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
)

const (
	TypeItemCollection      = "item_collection"
	TypeMonsterKilling      = "monster_killing"
	TypeCombinationPuzzle   = "combination_puzzle"
	TypeReactorTrigger      = "reactor_trigger"
	TypeWarpPuzzle          = "warp_puzzle"
	TypeSequenceMemoryGame  = "sequence_memory_game"
	TypeBonus               = "bonus"
	TypeBoss                = "boss"
)

type Model struct {
	index           uint32
	name            string
	mapIds          []uint32
	stageType       string
	duration        uint64
	clearConditions []condition.Model
	rewards         []reward.Model
	warpType        string
	properties      map[string]any
}

func (m Model) Index() uint32                       { return m.index }
func (m Model) Name() string                        { return m.name }
func (m Model) MapIds() []uint32                    { return m.mapIds }
func (m Model) Type() string                        { return m.stageType }
func (m Model) Duration() uint64                    { return m.duration }
func (m Model) ClearConditions() []condition.Model  { return m.clearConditions }
func (m Model) Rewards() []reward.Model             { return m.rewards }
func (m Model) WarpType() string                    { return m.warpType }
func (m Model) Properties() map[string]any          { return m.properties }

