package rates

// Builder constructs a rates Model for test setup.
type Builder struct {
	expRate      float64
	mesoRate     float64
	itemDropRate float64
	questExpRate float64
}

func NewBuilder() *Builder {
	return &Builder{
		expRate:      1.0,
		mesoRate:     1.0,
		itemDropRate: 1.0,
		questExpRate: 1.0,
	}
}

func (b *Builder) SetExpRate(expRate float64) *Builder {
	b.expRate = expRate
	return b
}

func (b *Builder) SetMesoRate(mesoRate float64) *Builder {
	b.mesoRate = mesoRate
	return b
}

func (b *Builder) SetItemDropRate(itemDropRate float64) *Builder {
	b.itemDropRate = itemDropRate
	return b
}

func (b *Builder) SetQuestExpRate(questExpRate float64) *Builder {
	b.questExpRate = questExpRate
	return b
}

func (b *Builder) Build() Model {
	return Model{
		expRate:      b.expRate,
		mesoRate:     b.mesoRate,
		itemDropRate: b.itemDropRate,
		questExpRate: b.questExpRate,
	}
}
