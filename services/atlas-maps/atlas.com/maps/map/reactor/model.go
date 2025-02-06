package reactor

import "strconv"

type Model struct {
	id        string
	name      string
	x         int16
	y         int16
	delay     uint32
	direction byte
}

func (m Model) Id() uint32 {
	id, _ := strconv.Atoi(m.id)
	return uint32(id)
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
