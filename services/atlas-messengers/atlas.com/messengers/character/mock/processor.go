package mock

import (
	"atlas-messengers/character"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	LoginFunc          func(transactionID uuid.UUID, field field.Model, characterId uint32) error
	LogoutFunc         func(transactionID uuid.UUID, characterId uint32) error
	ChannelChangeFunc  func(characterId uint32, channelId channel.Id) error
	JoinMessengerFunc  func(transactionID uuid.UUID, characterId uint32, messengerId uint32) error
	LeaveMessengerFunc func(transactionID uuid.UUID, characterId uint32) error
	GetByIdFunc        func(characterId uint32) (character.Model, error)
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Login(transactionID uuid.UUID, field field.Model, characterId uint32) error {
	if m.LoginFunc != nil {
		return m.LoginFunc(transactionID, field, characterId)
	}
	return nil
}

func (m *ProcessorMock) Logout(transactionID uuid.UUID, characterId uint32) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(transactionID, characterId)
	}
	return nil
}

func (m *ProcessorMock) ChannelChange(characterId uint32, channelId channel.Id) error {
	if m.ChannelChangeFunc != nil {
		return m.ChannelChangeFunc(characterId, channelId)
	}
	return nil
}

func (m *ProcessorMock) JoinMessenger(transactionID uuid.UUID, characterId uint32, messengerId uint32) error {
	if m.JoinMessengerFunc != nil {
		return m.JoinMessengerFunc(transactionID, characterId, messengerId)
	}
	return nil
}

func (m *ProcessorMock) LeaveMessenger(transactionID uuid.UUID, characterId uint32) error {
	if m.LeaveMessengerFunc != nil {
		return m.LeaveMessengerFunc(transactionID, characterId)
	}
	return nil
}

func (m *ProcessorMock) GetById(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}
