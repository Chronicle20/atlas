package position

type Builder struct {
	x int16
	y int16
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		x: b.x,
		y: b.y,
	}, nil
}
