package reactor

type Model struct {
	id             uint32
	classification uint32
	name           string
	x              int16
	y              int16
	delay          uint32
	direction      byte
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Classification() uint32 {
	return m.classification
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Delay() uint32 {
	return m.delay
}

func (m Model) Direction() byte {
	return m.direction
}
