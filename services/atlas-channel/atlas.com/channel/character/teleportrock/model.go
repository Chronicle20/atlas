package teleportrock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// Model is the channel-side read model of both saved-map lists (unpadded).
type Model struct {
	regular []_map.Id
	vip     []_map.Id
}

func NewModel(regular []_map.Id, vip []_map.Id) Model {
	return Model{regular: regular, vip: vip}
}

func (m Model) Regular() []_map.Id { return m.regular }
func (m Model) Vip() []_map.Id     { return m.vip }

func (m Model) List(vip bool) []_map.Id {
	if vip {
		return m.vip
	}
	return m.regular
}

func (m Model) Contains(vip bool, mapId _map.Id) bool {
	for _, v := range m.List(vip) {
		if v == mapId {
			return true
		}
	}
	return false
}
