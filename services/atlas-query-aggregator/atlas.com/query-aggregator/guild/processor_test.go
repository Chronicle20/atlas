package guild_test

import (
	"atlas-query-aggregator/guild"
	"atlas-query-aggregator/guild/member"
	"atlas-query-aggregator/guild/mock"
	"atlas-query-aggregator/guild/title"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

// createTestGuildModel is a helper to create guild models for testing
func createTestGuildModel(id uint32, leaderId uint32) guild.Model {
	rm := guild.RestModel{
		Id:       id,
		LeaderId: leaderId,
		Members:  []member.RestModel{},
		Titles:   []title.RestModel{},
	}
	guildModel, _ := guild.Extract(rm)
	return guildModel
}

func TestProcessorMock_GetByMemberId_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		GetByMemberIdFunc: func(decorators ...model.Decorator[guild.Model]) func(memberId uint32) (guild.Model, error) {
			return func(memberId uint32) (guild.Model, error) {
				g := createTestGuildModel(100, 123)
				for _, d := range decorators {
					g = d(g)
				}
				return g, nil
			}
		},
	}

	result, err := mockProcessor.GetByMemberId()(456)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Id() != 100 {
		t.Errorf("Expected Id=100, got %d", result.Id())
	}

	if result.LeaderId() != 123 {
		t.Errorf("Expected LeaderId=123, got %d", result.LeaderId())
	}
}

func TestProcessorMock_GetByMemberId_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		GetByMemberIdFunc: func(decorators ...model.Decorator[guild.Model]) func(memberId uint32) (guild.Model, error) {
			return func(memberId uint32) (guild.Model, error) {
				return guild.Model{}, errors.New("guild not found")
			}
		},
	}

	_, err := mockProcessor.GetByMemberId()(999)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "guild not found" {
		t.Errorf("Expected error message 'guild not found', got '%s'", err.Error())
	}
}

func TestProcessorMock_IsLeader_True(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		IsLeaderFunc: func(characterId uint32) (bool, error) {
			return characterId == 123, nil
		},
	}

	isLeader, err := mockProcessor.IsLeader(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !isLeader {
		t.Error("Expected character 123 to be a leader")
	}
}

func TestProcessorMock_IsLeader_False(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		IsLeaderFunc: func(characterId uint32) (bool, error) {
			return characterId == 123, nil
		},
	}

	isLeader, err := mockProcessor.IsLeader(456)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if isLeader {
		t.Error("Expected character 456 to not be a leader")
	}
}

func TestProcessorMock_IsLeader_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		IsLeaderFunc: func(characterId uint32) (bool, error) {
			return false, errors.New("service unavailable")
		},
	}

	_, err := mockProcessor.IsLeader(123)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_HasGuild_True(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		HasGuildFunc: func(characterId uint32) (bool, error) {
			return true, nil
		},
	}

	hasGuild, err := mockProcessor.HasGuild(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !hasGuild {
		t.Error("Expected character to have a guild")
	}
}

func TestProcessorMock_HasGuild_False(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{
		HasGuildFunc: func(characterId uint32) (bool, error) {
			return false, nil
		},
	}

	hasGuild, err := mockProcessor.HasGuild(456)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if hasGuild {
		t.Error("Expected character to not have a guild")
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorMock{}

	// Test default GetByMemberId returns empty model
	result, err := mockProcessor.GetByMemberId()(123)
	if err != nil {
		t.Errorf("Expected no error from default GetByMemberId, got %v", err)
	}

	if result.Id() != 0 {
		t.Errorf("Expected default Id=0, got %d", result.Id())
	}

	// Test default IsLeader returns false
	isLeader, err := mockProcessor.IsLeader(123)
	if err != nil {
		t.Errorf("Expected no error from default IsLeader, got %v", err)
	}

	if isLeader {
		t.Error("Expected default IsLeader to return false")
	}

	// Test default HasGuild returns false
	hasGuild, err := mockProcessor.HasGuild(123)
	if err != nil {
		t.Errorf("Expected no error from default HasGuild, got %v", err)
	}

	if hasGuild {
		t.Error("Expected default HasGuild to return false")
	}
}
