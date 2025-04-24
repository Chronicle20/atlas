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
