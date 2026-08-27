package consumable

// Builder constructs a Model in tests. Production callers use Extract.
type Builder struct {
	m Model
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetId(v uint32) *Builder             { b.m.id = v; return b }
func (b *Builder) SetCreate(v uint32) *Builder         { b.m.create = v; return b }
func (b *Builder) SetMonsterId(v uint32) *Builder      { b.m.monsterId = v; return b }
func (b *Builder) SetMonsterHp(v uint32) *Builder      { b.m.monsterHp = v; return b }
func (b *Builder) SetBridleProp(v uint32) *Builder     { b.m.bridleProp = v; return b }
func (b *Builder) SetBridlePropChg(v float64) *Builder { b.m.bridlePropChg = v; return b }
func (b *Builder) Build() Model                        { return b.m }
