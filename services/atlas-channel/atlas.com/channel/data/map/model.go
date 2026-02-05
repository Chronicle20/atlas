package map_

import _map "github.com/Chronicle20/atlas-constants/map"

type Model struct {
	clock       bool
	returnMapId _map.Id
	fieldLimit  uint32
	town        bool
}

func (m Model) Clock() bool {
	return m.clock
}

func (m Model) ReturnMapId() _map.Id {
	return m.returnMapId
}

func (m Model) FieldLimit() uint32 {
	return m.fieldLimit
}

func (m Model) Town() bool {
	return m.town
}

func (m Model) NoExpLossOnDeath() bool {
	return _map.NoExpLossOnDeath(m.fieldLimit)
}
