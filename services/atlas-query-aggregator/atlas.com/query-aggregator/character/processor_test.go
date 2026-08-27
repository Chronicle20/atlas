package character_test

import (
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/character/mock"
	"atlas-query-aggregator/guild"
	"atlas-query-aggregator/guild/member"
	guildMock "atlas-query-aggregator/guild/mock"
	"atlas-query-aggregator/guild/title"
	"atlas-query-aggregator/inventory"
	inventoryMock "atlas-query-aggregator/inventory/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// createTestGuild is a helper to create guild models for testing
func createTestGuild(id uint32, leaderId uint32) guild.Model {
	rm := guild.RestModel{
		Id:       id,
		LeaderId: leaderId,
		Members:  []member.RestModel{},
		Titles:   []title.RestModel{},
	}
	guildModel, _ := guild.Extract(rm)
	return guildModel
}

func TestProcessorImpl_GetById_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetByIdFunc: func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
			return func(characterId uint32) (character.Model, error) {
				m, err := character.NewBuilder().
					SetId(characterId).
					SetName("TestCharacter").
					SetLevel(50).
					SetJobId(100).
					SetMeso(10000).
					Build()
				if err != nil {
					return character.Model{}, err
				}
				for _, d := range decorators {
					m = d(m)
				}
				return m, nil
			}
		},
	}

	result, err := mockProcessor.GetById()(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Id() != 123 {
		t.Errorf("Expected Id=123, got %d", result.Id())
	}

	if result.Name() != "TestCharacter" {
		t.Errorf("Expected Name=TestCharacter, got %s", result.Name())
	}

	if result.Level() != 50 {
		t.Errorf("Expected Level=50, got %d", result.Level())
	}
}

func TestProcessorImpl_GetById_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetByIdFunc: func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
			return func(characterId uint32) (character.Model, error) {
				return character.Model{}, errors.New("character not found")
			}
		},
	}

	_, err := mockProcessor.GetById()(999)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "character not found" {
		t.Errorf("Expected error message 'character not found', got '%s'", err.Error())
	}
}

func TestProcessorImpl_GetById_WithDecorators(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetByIdFunc: func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
			return func(characterId uint32) (character.Model, error) {
				m, err := character.NewBuilder().
					SetId(characterId).
					SetLevel(50).
					Build()
				if err != nil {
					return character.Model{}, err
				}
				for _, d := range decorators {
					m = d(m)
				}
				return m, nil
			}
		},
	}

	levelDecorator := func(m character.Model) character.Model {
		decorated, err := character.Clone(m).SetLevel(100).Build()
		if err != nil {
			t.Fatalf("failed to build character: %v", err)
		}
		return decorated
	}

	result, err := mockProcessor.GetById(levelDecorator)(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Level() != 100 {
		t.Errorf("Expected Level=100 after decorator, got %d", result.Level())
	}
}

func TestProcessorImpl_InventoryDecorator(t *testing.T) {
	testInventory := inventory.NewBuilder(123).Build()

	mockProcessor := &mock.ProcessorImpl{
		InventoryDecoratorFunc: func(m character.Model) character.Model {
			decorated, err := character.Clone(m).SetInventory(testInventory).Build()
			if err != nil {
				t.Fatalf("failed to build character: %v", err)
			}
			return decorated
		},
	}

	inputModel, err := character.NewBuilder().SetId(123).Build()
	if err != nil {
		t.Fatalf("failed to build character: %v", err)
	}
	result := mockProcessor.InventoryDecorator(inputModel)

	if result.Id() != 123 {
		t.Errorf("Expected Id=123, got %d", result.Id())
	}
}

func TestProcessorImpl_GuildDecorator(t *testing.T) {
	testGuild := createTestGuild(1, 123)

	mockProcessor := &mock.ProcessorImpl{
		GuildDecoratorFunc: func(m character.Model) character.Model {
			decorated, err := character.Clone(m).SetGuild(testGuild).Build()
			if err != nil {
				t.Fatalf("failed to build character: %v", err)
			}
			return decorated
		},
	}

	inputModel, err := character.NewBuilder().SetId(123).Build()
	if err != nil {
		t.Fatalf("failed to build character: %v", err)
	}
	result := mockProcessor.GuildDecorator(inputModel)

	if result.Guild().Id() != 1 {
		t.Errorf("Expected Guild Id=1, got %d", result.Guild().Id())
	}

	if result.Guild().LeaderId() != 123 {
		t.Errorf("Expected Guild LeaderId=123, got %d", result.Guild().LeaderId())
	}
}

func TestProcessorImpl_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetById returns a model with the requested id
	result, err := mockProcessor.GetById()(123)
	if err != nil {
		t.Errorf("Expected no error from default GetById, got %v", err)
	}

	if result.Id() != 123 {
		t.Errorf("Expected default Id=123, got %d", result.Id())
	}

	// Test default InventoryDecorator returns input unchanged
	inputModel, err := character.NewBuilder().SetId(456).Build()
	if err != nil {
		t.Fatalf("failed to build character: %v", err)
	}
	decorated := mockProcessor.InventoryDecorator(inputModel)
	if decorated.Id() != 456 {
		t.Errorf("Expected InventoryDecorator to return input unchanged, got Id=%d", decorated.Id())
	}

	// Test default GuildDecorator returns input unchanged
	decorated = mockProcessor.GuildDecorator(inputModel)
	if decorated.Id() != 456 {
		t.Errorf("Expected GuildDecorator to return input unchanged, got Id=%d", decorated.Id())
	}
}

func TestProcessorImpl_IntegrationWithGuildMock(t *testing.T) {
	guildProcessor := &guildMock.ProcessorMock{
		GetByMemberIdFunc: func(decorators ...model.Decorator[guild.Model]) func(memberId uint32) (guild.Model, error) {
			return func(memberId uint32) (guild.Model, error) {
				return createTestGuild(100, memberId), nil
			}
		},
		IsLeaderFunc: func(characterId uint32) (bool, error) {
			return characterId == 123, nil
		},
	}

	// Test IsLeader
	isLeader, err := guildProcessor.IsLeader(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !isLeader {
		t.Error("Expected character 123 to be leader")
	}

	isLeader, err = guildProcessor.IsLeader(456)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if isLeader {
		t.Error("Expected character 456 to not be leader")
	}
}

func TestProcessorImpl_IntegrationWithInventoryMock(t *testing.T) {
	invProcessor := &inventoryMock.ProcessorImpl{
		GetByCharacterIdFunc: func(characterId uint32) (inventory.Model, error) {
			return inventory.NewBuilder(characterId).Build(), nil
		},
	}

	inv, err := invProcessor.GetByCharacterId(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if inv.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", inv.CharacterId())
	}
}
