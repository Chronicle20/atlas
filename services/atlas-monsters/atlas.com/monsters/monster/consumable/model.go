package consumable

// Model is the immutable catch-item view. Construct it from REST via Extract or
// in tests via NewModelBuilder.
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

type ModelBuilder struct {
	m Model
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func (b *ModelBuilder) SetId(v uint32) *ModelBuilder             { b.m.id = v; return b }
func (b *ModelBuilder) SetCreate(v uint32) *ModelBuilder         { b.m.create = v; return b }
func (b *ModelBuilder) SetMonsterId(v uint32) *ModelBuilder      { b.m.monsterId = v; return b }
func (b *ModelBuilder) SetMonsterHp(v uint32) *ModelBuilder      { b.m.monsterHp = v; return b }
func (b *ModelBuilder) SetBridleProp(v uint32) *ModelBuilder     { b.m.bridleProp = v; return b }
func (b *ModelBuilder) SetBridlePropChg(v float64) *ModelBuilder { b.m.bridlePropChg = v; return b }
func (b *ModelBuilder) Build() Model                             { return b.m }
