package mock

import (
	"atlas-guilds/party"

	"github.com/Chronicle20/atlas-model/model"
)

type ProcessorMock struct {
	GetByIdFunc          func(partyId uint32) (party.Model, error)
	GetByMemberIdFunc    func(memberId uint32) (party.Model, error)
	ByIdProviderFunc     func(partyId uint32) model.Provider[party.Model]
	ByMemberIdProviderFunc func(memberId uint32) model.Provider[[]party.Model]
}

func (m *ProcessorMock) GetById(partyId uint32) (party.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(partyId)
	}
	return party.Model{}, nil
}

func (m *ProcessorMock) GetByMemberId(memberId uint32) (party.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(memberId)
	}
	return party.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(partyId uint32) model.Provider[party.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(partyId)
	}
	return func() (party.Model, error) {
		return party.Model{}, nil
	}
}

func (m *ProcessorMock) ByMemberIdProvider(memberId uint32) model.Provider[[]party.Model] {
	if m.ByMemberIdProviderFunc != nil {
		return m.ByMemberIdProviderFunc(memberId)
	}
	return func() ([]party.Model, error) {
		return []party.Model{}, nil
	}
}
