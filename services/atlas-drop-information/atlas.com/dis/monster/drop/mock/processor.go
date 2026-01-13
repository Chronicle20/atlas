package mock

import (
	drop "atlas-drops-information/monster/drop"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorMock is a mock implementation of the drop.Processor interface
type ProcessorMock struct {
	GetAllFunc        func() model.Provider[[]drop.Model]
	GetForMonsterFunc func(monsterId uint32) model.Provider[[]drop.Model]
}

// GetAll is a mock implementation of the drop.Processor.GetAll method
func (m *ProcessorMock) GetAll() model.Provider[[]drop.Model] {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return model.FixedProvider[[]drop.Model](nil)
}

// GetForMonster is a mock implementation of the drop.Processor.GetForMonster method
func (m *ProcessorMock) GetForMonster(monsterId uint32) model.Provider[[]drop.Model] {
	if m.GetForMonsterFunc != nil {
		return m.GetForMonsterFunc(monsterId)
	}
	return model.FixedProvider[[]drop.Model](nil)
}
