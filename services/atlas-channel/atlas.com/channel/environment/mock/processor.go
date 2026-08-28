package mock

import (
	"atlas-channel/environment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetAllFunc func(f field.Model) ([]environment.RestModel, error)
}

var _ environment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAll(f field.Model) ([]environment.RestModel, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(f)
	}
	return []environment.RestModel{}, nil
}
