package mock

import (
	"atlas-inventory/data/equipment/slot"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdModelProviderFunc func(id uint32) model.Provider[[]slot.Model]
	GetByIdFunc           func(id uint32) ([]slot.Model, error)
}

var _ slot.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdModelProvider(id uint32) model.Provider[[]slot.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(id)
	}
	return model.FixedProvider([]slot.Model{})
}

func (m *ProcessorMock) GetById(id uint32) ([]slot.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return []slot.Model{}, nil
}
