package mock

import (
	_map "atlas-monster-death/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	CharacterIdsInFieldFunc func(f field.Model) ([]uint32, error)
}

var _ _map.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CharacterIdsInField(f field.Model) ([]uint32, error) {
	if m.CharacterIdsInFieldFunc != nil {
		return m.CharacterIdsInFieldFunc(f)
	}
	return []uint32{}, nil
}
