package position

type Model struct {
	id uint32
	x1 int16
	y1 int16
	x2 int16
	y2 int16
}

func (m Model) Id() uint32 {
	return m.id
}

func NewModel(id uint32, x1 int16, y1 int16, x2 int16, y2 int16) Model {
	return Model{
		id: id,
		x1: x1,
		y1: y1,
		x2: x2,
		y2: y2,
	}
}
