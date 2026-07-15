package mock

import (
	"atlas-channel/party"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	CreateFunc             func(characterId uint32) error
	LeaveFunc              func(partyId uint32, characterId uint32) error
	ExpelFunc              func(partyId uint32, characterId uint32, targetCharacterId uint32) error
	ChangeLeaderFunc       func(partyId uint32, characterId uint32, targetCharacterId uint32) error
	RequestInviteFunc      func(characterId uint32, targetCharacterId uint32) error
	GetByIdFunc            func(partyId uint32) (party.Model, error)
	ByIdProviderFunc       func(partyId uint32) model.Provider[party.Model]
	GetByMemberIdFunc      func(memberId uint32) (party.Model, error)
	ByMemberIdProviderFunc func(memberId uint32) model.Provider[party.Model]
}

var _ party.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(characterId uint32) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(characterId)
	}
	return nil
}

func (m *ProcessorMock) Leave(partyId uint32, characterId uint32) error {
	if m.LeaveFunc != nil {
		return m.LeaveFunc(partyId, characterId)
	}
	return nil
}

func (m *ProcessorMock) Expel(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	if m.ExpelFunc != nil {
		return m.ExpelFunc(partyId, characterId, targetCharacterId)
	}
	return nil
}

func (m *ProcessorMock) ChangeLeader(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	if m.ChangeLeaderFunc != nil {
		return m.ChangeLeaderFunc(partyId, characterId, targetCharacterId)
	}
	return nil
}

func (m *ProcessorMock) RequestInvite(characterId uint32, targetCharacterId uint32) error {
	if m.RequestInviteFunc != nil {
		return m.RequestInviteFunc(characterId, targetCharacterId)
	}
	return nil
}

func (m *ProcessorMock) GetById(partyId uint32) (party.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(partyId)
	}
	return party.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(partyId uint32) model.Provider[party.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(partyId)
	}
	return model.FixedProvider(party.Model{})
}

func (m *ProcessorMock) GetByMemberId(memberId uint32) (party.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(memberId)
	}
	return party.Model{}, nil
}

func (m *ProcessorMock) ByMemberIdProvider(memberId uint32) model.Provider[party.Model] {
	if m.ByMemberIdProviderFunc != nil {
		return m.ByMemberIdProviderFunc(memberId)
	}
	return model.FixedProvider(party.Model{})
}
