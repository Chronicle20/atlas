package character

// Builder constructs a Model.
type Builder struct {
	id    uint32
	level byte
	meso  uint32
}

func NewBuilder(id uint32) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetLevel(level byte) *Builder {
	b.level = level
	return b
}

func (b *Builder) SetMeso(meso uint32) *Builder {
	b.meso = meso
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		id:    b.id,
		level: b.level,
		meso:  b.meso,
	}, nil
}
