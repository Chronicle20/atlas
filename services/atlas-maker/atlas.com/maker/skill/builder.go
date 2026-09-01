package skill

// Builder constructs a Model.
type Builder struct {
	id    uint32
	level byte
}

func NewBuilder(id uint32) *Builder {
	return &Builder{id: id}
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
