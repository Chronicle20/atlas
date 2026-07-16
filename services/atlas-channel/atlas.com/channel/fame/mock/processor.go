package mock

import (
	"atlas-channel/fame"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	RequestChangeFunc func(f field.Model, characterId uint32, targetId uint32, amount int8) error
}

var _ fame.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestChange(f field.Model, characterId uint32, targetId uint32, amount int8) error {
	if m.RequestChangeFunc != nil {
		return m.RequestChangeFunc(f, characterId, targetId, amount)
	}
	return nil
}
