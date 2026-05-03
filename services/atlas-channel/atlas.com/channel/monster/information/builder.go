package information

// ModelBuilder builds Model instances for tests.
type ModelBuilder struct {
	monsterId uint32
	attacks   []AttackInfo
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func (b *ModelBuilder) SetMonsterId(id uint32) *ModelBuilder {
	b.monsterId = id
	return b
}

func (b *ModelBuilder) SetAttacks(attacks []AttackInfo) *ModelBuilder {
	b.attacks = attacks
	return b
}

func (b *ModelBuilder) Build() Model {
	attacks := b.attacks
	if attacks == nil {
		attacks = []AttackInfo{}
	}
	return Model{
		monsterId: b.monsterId,
		attacks:   attacks,
	}
}
