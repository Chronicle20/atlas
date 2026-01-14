package character

type Builder struct {
	id    uint32
	level byte
}

func NewBuilder() *Builder {
	return &Builder{
		level: 1,
	}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetLevel(level byte) *Builder {
	b.level = level
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		id:    b.id,
		level: b.level,
	}, nil
}
