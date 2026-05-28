package mock

import (
	"atlas-messages/data/foothold"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Processor struct {
	GetBelowFn func(mapId _map.Id, x int16, y int16) (foothold.Model, error)
}

func (m *Processor) GetBelow(mapId _map.Id, x int16, y int16) (foothold.Model, error) {
	if m.GetBelowFn != nil {
		return m.GetBelowFn(mapId, x, y)
	}
	return foothold.Model{}, nil
}
