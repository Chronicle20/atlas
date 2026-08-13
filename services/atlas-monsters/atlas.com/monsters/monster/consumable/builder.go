package consumable

// ModelBuilder constructs a Model in tests. Production callers use Extract.
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
