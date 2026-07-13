package mock

import (
	"atlas-doors/party"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

type ProcessorMock struct {
	GetByMemberIdFunc func(characterId character.Id) (party.Model, error)
	GetByIdFunc       func(partyId uint32) (party.Model, error)
}

var _ party.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByMemberId(characterId character.Id) (party.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(characterId)
	}
	return party.Model{}, nil
}

func (m *ProcessorMock) GetById(partyId uint32) (party.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(partyId)
	}
	return party.Model{}, nil
}
