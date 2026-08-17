package mock

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

// ProcessorMock is a mock implementation of the guild.Processor interface.
type ProcessorMock struct {
	RequestNameFunc             func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestEmblemFunc           func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestDisbandFunc          func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestCapacityIncreaseFunc func(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestLeaveFunc            func(transactionId uuid.UUID, characterId uint32, guildId uint32, force bool) error
	RequestRejoinFunc           func(transactionId uuid.UUID, characterId uint32, guildId uint32, title byte) error
}

func (m *ProcessorMock) RequestName(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	if m.RequestNameFunc != nil {
		return m.RequestNameFunc(transactionId, ch, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestEmblem(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	if m.RequestEmblemFunc != nil {
		return m.RequestEmblemFunc(transactionId, ch, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestDisband(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	if m.RequestDisbandFunc != nil {
		return m.RequestDisbandFunc(transactionId, ch, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestCapacityIncrease(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	if m.RequestCapacityIncreaseFunc != nil {
		return m.RequestCapacityIncreaseFunc(transactionId, ch, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestLeave(transactionId uuid.UUID, characterId uint32, guildId uint32, force bool) error {
	if m.RequestLeaveFunc != nil {
		return m.RequestLeaveFunc(transactionId, characterId, guildId, force)
	}
	return nil
}

func (m *ProcessorMock) RequestRejoin(transactionId uuid.UUID, characterId uint32, guildId uint32, title byte) error {
	if m.RequestRejoinFunc != nil {
		return m.RequestRejoinFunc(transactionId, characterId, guildId, title)
	}
	return nil
}
