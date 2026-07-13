package mock

import (
	"atlas-monster-death/monster/drop/position"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetInMapFunc func(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[position.Model]
}

var _ position.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetInMap(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) model.Provider[position.Model] {
	if m.GetInMapFunc != nil {
		return m.GetInMapFunc(mapId, initialX, initialY, fallbackX, fallbackY)
	}
	return model.FixedProvider(position.Model{})
}
