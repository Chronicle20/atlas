package mock

import (
	"atlas-inventory/data/setup"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdModelProviderFunc func(id uint32) model.Provider[setup.Model]
	GetByIdFunc           func(id uint32) (setup.Model, error)
}

var _ setup.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdModelProvider(id uint32) model.Provider[setup.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(id)
	}
	return model.FixedProvider(setup.Model{})
}

func (m *ProcessorMock) GetById(id uint32) (setup.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return setup.Model{}, nil
}
