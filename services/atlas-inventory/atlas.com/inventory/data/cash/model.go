package cash

// Model is a cash item template's expiration-extender attributes, as resolved
// from atlas-data.
type Model struct {
	id      uint32
	addTime uint32
	maxDays uint32
}

func (m Model) Id() uint32 { return m.id }

// AddTime is the expiration grant in SECONDS.
func (m Model) AddTime() uint32 { return m.addTime }

// MaxDays is the ceiling in DAYS, anchored to now.
func (m Model) MaxDays() uint32 { return m.maxDays }

// ModelBuilder constructs a Model. It exists so callers outside this package
// (notably test doubles) can build one without exported fields, per the
// project's Builder convention.
type ModelBuilder struct {
	id      uint32
	addTime uint32
	maxDays uint32
}

func NewModelBuilder(id uint32) *ModelBuilder {
	return &ModelBuilder{id: id}
}

func (b *ModelBuilder) SetAddTime(v uint32) *ModelBuilder { b.addTime = v; return b }
func (b *ModelBuilder) SetMaxDays(v uint32) *ModelBuilder { b.maxDays = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{id: b.id, addTime: b.addTime, maxDays: b.maxDays}
}
