package point

type Model struct {
	x X
	y Y
}

func NewModel(x X, y Y) Model {
	return Model{x: x, y: y}
}

func (m Model) X() X {
	return m.x
}

func (m Model) Y() Y {
	return m.y
}
