package mock

import (
	"atlas-login/guild"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetByMemberIdFunc      func(memberId uint32) (guild.Model, error)
	ByMemberIdProviderFunc func(memberId uint32) model.Provider[[]guild.Model]
	IsGuildMasterFunc      func(characterId uint32) (bool, error)
}

var _ guild.Processor = (*ProcessorMock)(nil)

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
	return model.FixedProvider([]guild.Model{})
}

func (m *ProcessorMock) IsGuildMaster(characterId uint32) (bool, error) {
	if m.IsGuildMasterFunc != nil {
		return m.IsGuildMasterFunc(characterId)
	}
	return false, nil
}
