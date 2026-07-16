package mock

import (
	"atlas-inventory/data/etc"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdModelProviderFunc func(id uint32) model.Provider[etc.Model]
	GetByIdFunc           func(id uint32) (etc.Model, error)
}

var _ etc.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdModelProvider(id uint32) model.Provider[etc.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(id)
	}
	return model.FixedProvider(etc.Model{})
}

func (m *ProcessorMock) GetById(id uint32) (etc.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return etc.Model{}, nil
}
