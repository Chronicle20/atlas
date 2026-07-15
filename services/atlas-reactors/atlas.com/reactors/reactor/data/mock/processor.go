package mock

import (
	"atlas-reactors/reactor/data"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdProviderFunc func(id uint32) model.Provider[data.Model]
	GetByIdFunc      func(id uint32) (data.Model, error)
}

var _ data.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(id uint32) model.Provider[data.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(data.Model{})
}

func (m *ProcessorMock) GetById(id uint32) (data.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return data.Model{}, nil
}
