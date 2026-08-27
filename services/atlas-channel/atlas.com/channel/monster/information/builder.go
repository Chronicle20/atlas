package information

// Builder builds Model instances for tests.
type Builder struct {
	monsterId uint32
	attacks   []AttackInfo
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetMonsterId(id uint32) *Builder {
	b.monsterId = id
	return b
}

func (b *Builder) SetAttacks(attacks []AttackInfo) *Builder {
	b.attacks = attacks
	return b
}

func (b *Builder) Build() Model {
	attacks := b.attacks
	if attacks == nil {
		attacks = []AttackInfo{}
	}
	return Model{
		monsterId: b.monsterId,
		attacks:   attacks,
	}
}
