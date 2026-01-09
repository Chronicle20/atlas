package medal

// Model represents a visited map for a medal quest
type Model struct {
	id    uint32
	mapId uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) MapId() uint32 {
	return m.mapId
}
