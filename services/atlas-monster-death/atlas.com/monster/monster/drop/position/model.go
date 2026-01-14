package position

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

func NewModel(x int16, y int16) *Model {
	return &Model{
		x: x,
		y: y,
	}
}
