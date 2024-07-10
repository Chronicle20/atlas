package point

type Model struct {
	x int16
	y int16
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m *Model) SetX(value int16) *Model {
	return NewModel(value, m.Y())
}

func NewModel(x int16, y int16) *Model {
	return &Model{
		x: x,
		y: y,
	}
}
