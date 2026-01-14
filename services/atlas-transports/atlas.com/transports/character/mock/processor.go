package mock

import (
	"atlas-transports/character"
	"atlas-transports/kafka/message"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
)

// Compile-time interface compliance check
var _ character.Processor = (*ProcessorMock)(nil)

// ProcessorMock is a mock implementation of the character.Processor interface
type ProcessorMock struct {
	WarpRandomFunc        func(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error
	WarpRandomAndEmitFunc func(characterId uint32, fieldId field.Id) error
	WarpToPortalFunc      func(mb *message.Buffer) func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error
}

// WarpRandom is a mock implementation
func (m *ProcessorMock) WarpRandom(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
	if m.WarpRandomFunc != nil {
		return m.WarpRandomFunc(mb)
	}
	return func(characterId uint32) func(fieldId field.Id) error {
		return func(fieldId field.Id) error {
			return nil
		}
	}
}

// WarpRandomAndEmit is a mock implementation
func (m *ProcessorMock) WarpRandomAndEmit(characterId uint32, fieldId field.Id) error {
	if m.WarpRandomAndEmitFunc != nil {
		return m.WarpRandomAndEmitFunc(characterId, fieldId)
	}
	return nil
}

// WarpToPortal is a mock implementation
func (m *ProcessorMock) WarpToPortal(mb *message.Buffer) func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error {
	if m.WarpToPortalFunc != nil {
		return m.WarpToPortalFunc(mb)
	}
	return func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error {
		return nil
	}
}
