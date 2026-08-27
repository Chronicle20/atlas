package consumable

// Model is the immutable catch-item view. Construct it from REST via Extract or
// in tests via NewBuilder.
type Model struct {
	id            uint32
	create        uint32
	monsterId     uint32
	monsterHp     uint32
	bridleProp    uint32
	bridlePropChg float64
}

func (m Model) Id() uint32             { return m.id }
func (m Model) Create() uint32         { return m.create }
func (m Model) MonsterId() uint32      { return m.monsterId }
func (m Model) MonsterHp() uint32      { return m.monsterHp }
func (m Model) BridleProp() uint32     { return m.bridleProp }
func (m Model) BridlePropChg() float64 { return m.bridlePropChg }
