package mock

import (
	"atlas-monster-death/data/equipment/statistics"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdModelProviderFunc func(id uint32) model.Provider[statistics.Model]
	GetByIdFunc           func(id uint32) (statistics.Model, error)
}

var _ statistics.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdModelProvider(id uint32) model.Provider[statistics.Model] {
	if m.ByIdModelProviderFunc != nil {
		return m.ByIdModelProviderFunc(id)
	}
	return model.FixedProvider(statistics.Model{})
}

func (m *ProcessorMock) GetById(id uint32) (statistics.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return statistics.Model{}, nil
}
