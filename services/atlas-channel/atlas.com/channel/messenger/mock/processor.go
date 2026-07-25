package mock

import (
	"atlas-channel/messenger"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	CreateFunc             func(characterId uint32) error
	LeaveFunc              func(messengerId uint32, characterId uint32) error
	RequestInviteFunc      func(characterId uint32, targetCharacterId uint32) error
	GetByIdFunc            func(messengerId uint32) (messenger.Model, error)
	ByIdProviderFunc       func(messengerId uint32) model.Provider[messenger.Model]
	GetByMemberIdFunc      func(memberId uint32) (messenger.Model, error)
	ByMemberIdProviderFunc func(memberId uint32) model.Provider[messenger.Model]
}

var _ messenger.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(characterId uint32) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(characterId)
	}
	return nil
}

func (m *ProcessorMock) Leave(messengerId uint32, characterId uint32) error {
	if m.LeaveFunc != nil {
		return m.LeaveFunc(messengerId, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestInvite(characterId uint32, targetCharacterId uint32) error {
	if m.RequestInviteFunc != nil {
		return m.RequestInviteFunc(characterId, targetCharacterId)
	}
	return nil
}

func (m *ProcessorMock) GetById(messengerId uint32) (messenger.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(messengerId)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(messengerId uint32) model.Provider[messenger.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(messengerId)
	}
	return model.FixedProvider(messenger.Model{})
}

func (m *ProcessorMock) GetByMemberId(memberId uint32) (messenger.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(memberId)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) ByMemberIdProvider(memberId uint32) model.Provider[messenger.Model] {
	if m.ByMemberIdProviderFunc != nil {
		return m.ByMemberIdProviderFunc(memberId)
	}
	return model.FixedProvider(messenger.Model{})
}
