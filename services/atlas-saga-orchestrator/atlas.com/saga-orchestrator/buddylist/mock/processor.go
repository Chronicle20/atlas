package mock

import (
	"atlas-saga-orchestrator/kafka/message"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is a mock implementation of the buddylist.Processor interface.
type ProcessorMock struct {
	IncreaseCapacityAndEmitFunc func(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error
	IncreaseCapacityFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error
	RequestDeleteAndEmitFunc    func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
	RequestDeleteFunc           func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
	RestoreAndEmitFunc          func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
	RestoreFunc                 func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
}

func (m *ProcessorMock) IncreaseCapacityAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error {
	if m.IncreaseCapacityAndEmitFunc != nil {
		return m.IncreaseCapacityAndEmitFunc(transactionId, characterId, worldId, newCapacity)
	}
	return nil
}

func (m *ProcessorMock) IncreaseCapacity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error {
	if m.IncreaseCapacityFunc != nil {
		return m.IncreaseCapacityFunc(mb)
	}
	return func(uuid.UUID, uint32, world.Id, byte) error { return nil }
}

func (m *ProcessorMock) RequestDeleteAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	if m.RequestDeleteAndEmitFunc != nil {
		return m.RequestDeleteAndEmitFunc(transactionId, characterId, worldId, targetId)
	}
	return nil
}

func (m *ProcessorMock) RequestDelete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	if m.RequestDeleteFunc != nil {
		return m.RequestDeleteFunc(mb)
	}
	return func(uuid.UUID, uint32, world.Id, uint32) error { return nil }
}

func (m *ProcessorMock) RestoreAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	if m.RestoreAndEmitFunc != nil {
		return m.RestoreAndEmitFunc(transactionId, characterId, worldId, targetId)
	}
	return nil
}

func (m *ProcessorMock) Restore(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(mb)
	}
	return func(uuid.UUID, uint32, world.Id, uint32) error { return nil }
}
