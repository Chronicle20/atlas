package mock

import (
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the compartment.Processor interface
type ProcessorMock struct {
	RequestCreateItemFunc  func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error
	RequestDestroyItemFunc func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error
}

// RequestCreateItem is a mock implementation of the compartment.Processor.RequestCreateItem method
func (m *ProcessorMock) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
	if m.RequestCreateItemFunc != nil {
		return m.RequestCreateItemFunc(transactionId, characterId, templateId, quantity)
	}
	return nil
}

// RequestDestroyItem is a mock implementation of the compartment.Processor.RequestDestroyItem method
func (m *ProcessorMock) RequestDestroyItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
	if m.RequestDestroyItemFunc != nil {
		return m.RequestDestroyItemFunc(transactionId, characterId, templateId, quantity)
	}
	return nil
}
