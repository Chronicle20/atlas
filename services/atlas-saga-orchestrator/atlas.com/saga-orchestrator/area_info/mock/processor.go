package mock

import (
	"atlas-saga-orchestrator/area_info"
)

// ProcessorMock is a mock implementation of the area_info.Processor interface.
type ProcessorMock struct {
	PutFunc func(characterId uint32, area uint16, info string) error
}

var _ area_info.Processor = (*ProcessorMock)(nil)

// Put is a mock implementation of the area_info.Processor.Put method.
func (m *ProcessorMock) Put(characterId uint32, area uint16, info string) error {
	if m.PutFunc != nil {
		return m.PutFunc(characterId, area, info)
	}
	return nil
}
