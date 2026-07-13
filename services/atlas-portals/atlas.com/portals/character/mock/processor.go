package mock

import (
	"atlas-portals/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	EnableActionsFunc func(f field.Model, characterId uint32)
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) EnableActions(f field.Model, characterId uint32) {
	if m.EnableActionsFunc != nil {
		m.EnableActionsFunc(f, characterId)
	}
}
