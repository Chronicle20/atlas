package cash

// Builder constructs a Model. It exists so callers outside this package
// (notably test doubles) can build one without exported fields, per the
// project's Builder convention.
type Builder struct {
	id      uint32
	addTime uint32
	maxDays uint32
	life    uint32
}

func NewBuilder(id uint32) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetAddTime(v uint32) *Builder { b.addTime = v; return b }
func (b *Builder) SetMaxDays(v uint32) *Builder { b.maxDays = v; return b }
func (b *Builder) SetLife(v uint32) *Builder    { b.life = v; return b }

func (b *Builder) Build() Model {
	return Model{id: b.id, addTime: b.addTime, maxDays: b.maxDays, life: b.life}
}
