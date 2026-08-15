package mock

import (
	"atlas-saga-orchestrator/kafka/message"
	"atlas-saga-orchestrator/pet"

	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the pet.Processor interface
type ProcessorMock struct {
	GainClosenessAndEmitFunc func(transactionId uuid.UUID, petId uint32, amount uint16) error
	GainClosenessFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error
	EvolveAndEmitFunc        func(transactionId uuid.UUID, petId uint32) error
	EvolveFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error
	RenameAndEmitFunc        func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error
	RenameFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error
	ReviveAndEmitFunc        func(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error
	ReviveFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error
}

var _ pet.Processor = (*ProcessorMock)(nil)

// GainClosenessAndEmit is a mock implementation of the pet.Processor.GainClosenessAndEmit method
func (m *ProcessorMock) GainClosenessAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error {
	if m.GainClosenessAndEmitFunc != nil {
		return m.GainClosenessAndEmitFunc(transactionId, petId, amount)
	}
	return nil
}

// GainCloseness is a mock implementation of the pet.Processor.GainCloseness method
func (m *ProcessorMock) GainCloseness(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error {
	if m.GainClosenessFunc != nil {
		return m.GainClosenessFunc(mb)
	}
	return func(uuid.UUID, uint32, uint16) error { return nil }
}

// EvolveAndEmit is a mock implementation of the pet.Processor.EvolveAndEmit method
func (m *ProcessorMock) EvolveAndEmit(transactionId uuid.UUID, petId uint32) error {
	if m.EvolveAndEmitFunc != nil {
		return m.EvolveAndEmitFunc(transactionId, petId)
	}
	return nil
}

// Evolve is a mock implementation of the pet.Processor.Evolve method
func (m *ProcessorMock) Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error {
	if m.EvolveFunc != nil {
		return m.EvolveFunc(mb)
	}
	return func(uuid.UUID, uint32) error { return nil }
}

// RenameAndEmit is a mock implementation of the pet.Processor.RenameAndEmit method
func (m *ProcessorMock) RenameAndEmit(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
	if m.RenameAndEmitFunc != nil {
		return m.RenameAndEmitFunc(transactionId, petId, characterId, name)
	}
	return nil
}

// Rename is a mock implementation of the pet.Processor.Rename method
func (m *ProcessorMock) Rename(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, characterId uint32, name string) error {
	if m.RenameFunc != nil {
		return m.RenameFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, string) error { return nil }
}

// ReviveAndEmit is a mock implementation of the pet.Processor.ReviveAndEmit method
func (m *ProcessorMock) ReviveAndEmit(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error {
	if m.ReviveAndEmitFunc != nil {
		return m.ReviveAndEmitFunc(transactionId, characterId, petId, sourceTemplateId)
	}
	return nil
}

// Revive is a mock implementation of the pet.Processor.Revive method
func (m *ProcessorMock) Revive(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error {
	if m.ReviveFunc != nil {
		return m.ReviveFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, uint32) error { return nil }
}
