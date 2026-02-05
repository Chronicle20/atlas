package mock

import (
	"atlas-pets/data/position"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
)

type Processor struct {
	GetBelowFn func(mapId _map.Id, x int16, y int16) model.Provider[position.Model]
}

func (m *Processor) GetBelow(mapId _map.Id, x int16, y int16) model.Provider[position.Model] {
	if m.GetBelowFn != nil {
		return m.GetBelowFn(mapId, x, y)
	}
	return nil
}
