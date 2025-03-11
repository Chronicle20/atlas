package character

type Model struct {
	id     uint32
	mapId  uint32
	x      int16
	y      int16
	stance byte
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Stance() byte {
	return m.stance
}

func (m Model) MapId() uint32 {
	return m.mapId
}
