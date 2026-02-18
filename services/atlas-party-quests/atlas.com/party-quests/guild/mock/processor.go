package mock

import (
	"atlas-party-quests/guild"

	"github.com/Chronicle20/atlas-model/model"
)

type ProcessorMock struct {
	GetByMemberIdFunc      func(memberId uint32) (guild.Model, error)
	ByMemberIdProviderFunc func(memberId uint32) model.Provider[[]guild.Model]
}

func (m *ProcessorMock) GetByMemberId(memberId uint32) (guild.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(memberId)
	}
	return guild.Model{}, nil
}

func (m *ProcessorMock) ByMemberIdProvider(memberId uint32) model.Provider[[]guild.Model] {
	if m.ByMemberIdProviderFunc != nil {
		return m.ByMemberIdProviderFunc(memberId)
	}
	return func() ([]guild.Model, error) {
		return []guild.Model{}, nil
	}
}
