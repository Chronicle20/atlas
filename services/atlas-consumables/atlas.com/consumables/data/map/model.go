package _map

import _map "github.com/Chronicle20/atlas-constants/map"

type Model struct {
	returnMapId _map.Id
}

func (m Model) ReturnMapId() _map.Id {
	return m.returnMapId
}
