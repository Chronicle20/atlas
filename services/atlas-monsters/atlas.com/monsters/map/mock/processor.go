package mock

import (
	_map "atlas-monsters/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	CharacterIdsInFieldProviderFunc func(f field.Model) model.Provider[[]uint32]
}

var _ _map.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32] {
	if m.CharacterIdsInFieldProviderFunc != nil {
		return m.CharacterIdsInFieldProviderFunc(f)
	}
	return model.FixedProvider([]uint32{})
}
