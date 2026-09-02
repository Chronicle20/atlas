package mock

import (
	"atlas-saga-orchestrator/playernpc"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is a mock implementation of the playernpc.Processor interface.
type ProcessorMock struct {
	GetCurrentLocationFunc func(characterId uint32) (world.Id, _map.Id, channel.Id, error)
}

var _ playernpc.Processor = (*ProcessorMock)(nil)

// GetCurrentLocation is a mock implementation of the
// playernpc.Processor.GetCurrentLocation method.
func (m *ProcessorMock) GetCurrentLocation(characterId uint32) (world.Id, _map.Id, channel.Id, error) {
	if m.GetCurrentLocationFunc != nil {
		return m.GetCurrentLocationFunc(characterId)
	}
	return 0, 0, 0, nil
}
