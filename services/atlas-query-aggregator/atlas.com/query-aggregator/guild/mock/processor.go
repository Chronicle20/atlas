package mock

import (
	"atlas-query-aggregator/guild"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorMock is a mock implementation of the guild.Processor interface
type ProcessorMock struct {
	GetByMemberIdFunc func(decorators ...model.Decorator[guild.Model]) func(memberId uint32) (guild.Model, error)
	IsLeaderFunc      func(characterId uint32) (bool, error)
	HasGuildFunc      func(characterId uint32) (bool, error)
}

// GetByMemberId mocks the GetByMemberId method
func (m *ProcessorMock) GetByMemberId(decorators ...model.Decorator[guild.Model]) func(memberId uint32) (guild.Model, error) {
	if m.GetByMemberIdFunc != nil {
		return m.GetByMemberIdFunc(decorators...)
	}
	return func(memberId uint32) (guild.Model, error) {
		return guild.Model{}, nil
	}
}

// IsLeader mocks the IsLeader method
func (m *ProcessorMock) IsLeader(characterId uint32) (bool, error) {
	if m.IsLeaderFunc != nil {
		return m.IsLeaderFunc(characterId)
	}
	return false, nil
}

// HasGuild mocks the HasGuild method
func (m *ProcessorMock) HasGuild(characterId uint32) (bool, error) {
	if m.HasGuildFunc != nil {
		return m.HasGuildFunc(characterId)
	}
	return false, nil
}
