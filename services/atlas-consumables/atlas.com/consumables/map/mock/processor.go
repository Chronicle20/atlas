package mock

import (
	_map "atlas-consumables/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	WarpRandomFunc   func(f field.Model) func(characterId uint32) error
	WarpToPortalFunc func(f field.Model, characterId uint32, pp model.Provider[uint32]) error
}

var _ _map.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WarpRandom(f field.Model) func(characterId uint32) error {
	if m.WarpRandomFunc != nil {
		return m.WarpRandomFunc(f)
	}
	return func(characterId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) WarpToPortal(f field.Model, characterId uint32, pp model.Provider[uint32]) error {
	if m.WarpToPortalFunc != nil {
		return m.WarpToPortalFunc(f, characterId, pp)
	}
	return nil
}
