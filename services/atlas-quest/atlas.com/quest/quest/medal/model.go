package medal

import _map "github.com/Chronicle20/atlas-constants/map"

// Model represents a visited map for a medal quest
type Model struct {
	id    uint32
	mapId _map.Id
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) MapId() _map.Id {
	return m.mapId
}
