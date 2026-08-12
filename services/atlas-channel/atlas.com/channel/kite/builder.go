package kite

// Builder constructs a Model. Tests and callers use it instead of a
// test-only constructor to keep the immutable-model pattern consistent.
type Builder struct {
	id          uint32
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
}

func NewBuilder(id uint32, characterId uint32) *Builder {
	return &Builder{
		id:          id,
		characterId: characterId,
	}
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

func (b *Builder) SetPosition(x int16, y int16) *Builder {
	b.x = x
	b.y = y
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:          b.id,
		characterId: b.characterId,
		name:        b.name,
		templateId:  b.templateId,
		message:     b.message,
		x:           b.x,
		y:           b.y,
	}
}
