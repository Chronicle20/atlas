package mock

import (
	drop "atlas-drops-information/reactor/drop"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorMock is a mock implementation of the drop.Processor interface
type ProcessorMock struct {
	GetAllFunc       func() model.Provider[[]drop.Model]
	GetForReactorFunc func(reactorId uint32) model.Provider[[]drop.Model]
}

// GetAll is a mock implementation of the drop.Processor.GetAll method
func (m *ProcessorMock) GetAll() model.Provider[[]drop.Model] {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return model.FixedProvider[[]drop.Model](nil)
}

// GetForReactor is a mock implementation of the drop.Processor.GetForReactor method
func (m *ProcessorMock) GetForReactor(reactorId uint32) model.Provider[[]drop.Model] {
	if m.GetForReactorFunc != nil {
		return m.GetForReactorFunc(reactorId)
	}
	return model.FixedProvider[[]drop.Model](nil)
}
