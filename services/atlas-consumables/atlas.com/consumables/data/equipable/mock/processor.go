package mock

import (
	"atlas-consumables/data/equipable"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdModelProviderFunc func(id uint32) model.Provider[equipable.Model]
	GetByIdFunc           func(id uint32) (equipable.Model, error)
}

var _ equipable.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdModelProvider(id uint32) model.Provider[equipable.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(id)
	}
	return model.FixedProvider(equipable.Model{})
}

func (m *ProcessorMock) GetById(id uint32) (equipable.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return equipable.Model{}, nil
}
