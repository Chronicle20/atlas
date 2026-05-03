package info

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

const noForcedReturnMapId = _map.Id(999999999)

type Model struct {
	id                _map.Id
	timeLimit         int32
	forcedReturnMapId _map.Id
}

func (m Model) Id() _map.Id {
	return m.id
}

func (m Model) TimeLimit() int32 {
	return m.timeLimit
}

func (m Model) ForcedReturnMapId() _map.Id {
	return m.forcedReturnMapId
}

func (m Model) IsTimeLimited() bool {
	return m.timeLimit > 0 && m.forcedReturnMapId != noForcedReturnMapId
}
