package mock

import (
	"atlas-channel/backeffect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetActiveFunc func(f field.Model) ([]backeffect.RestModel, error)
}

var _ backeffect.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetActive(f field.Model) ([]backeffect.RestModel, error) {
	if m.GetActiveFunc != nil {
		return m.GetActiveFunc(f)
	}
	return nil, nil
}
