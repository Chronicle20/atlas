package cash

// ModelBuilder constructs a Model. It exists so callers outside this package
// (notably test doubles) can build one without exported fields, per the
// project's Builder convention.
type ModelBuilder struct {
	id      uint32
	addTime uint32
	maxDays uint32
	life    uint32
}

func NewModelBuilder(id uint32) *ModelBuilder {
	return &ModelBuilder{id: id}
}

func (b *ModelBuilder) SetAddTime(v uint32) *ModelBuilder { b.addTime = v; return b }
func (b *ModelBuilder) SetMaxDays(v uint32) *ModelBuilder { b.maxDays = v; return b }
func (b *ModelBuilder) SetLife(v uint32) *ModelBuilder    { b.life = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{id: b.id, addTime: b.addTime, maxDays: b.maxDays, life: b.life}
}
