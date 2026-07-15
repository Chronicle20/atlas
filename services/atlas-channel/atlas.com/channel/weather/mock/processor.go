package mock

import (
	"atlas-channel/weather"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetActiveFunc func(f field.Model) (weather.RestModel, error)
}

var _ weather.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetActive(f field.Model) (weather.RestModel, error) {
	if m.GetActiveFunc != nil {
		return m.GetActiveFunc(f)
	}
	return weather.RestModel{}, nil
}
