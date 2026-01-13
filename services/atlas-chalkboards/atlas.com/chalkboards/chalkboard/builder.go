package chalkboard

type Builder struct {
	id      uint32
	message string
}

func NewBuilder(id uint32) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:      b.id,
		message: b.message,
	}
}
