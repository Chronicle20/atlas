package progress

type modelBuilder struct {
	id         uint32
	infoNumber uint32
	progress   string
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:         m.id,
		infoNumber: m.infoNumber,
		progress:   m.progress,
	}
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetInfoNumber(infoNumber uint32) *modelBuilder {
	b.infoNumber = infoNumber
	return b
}

func (b *modelBuilder) SetProgress(progress string) *modelBuilder {
	b.progress = progress
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		id:         b.id,
		infoNumber: b.infoNumber,
		progress:   b.progress,
	}
}
