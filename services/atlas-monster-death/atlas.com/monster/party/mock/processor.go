package mock

import (
	"atlas-monster-death/party"
)

type ProcessorMock struct {
	GetByMemberIdFunc func(memberId uint32) (party.Model, error)
}

var _ party.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByMemberId(memberId uint32) (party.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(memberId)
	}
	return party.Model{}, nil
}
