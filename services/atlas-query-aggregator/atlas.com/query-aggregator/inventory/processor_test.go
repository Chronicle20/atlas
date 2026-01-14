package inventory_test

import (
	"atlas-query-aggregator/inventory"
	"atlas-query-aggregator/inventory/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

func TestProcessorMock_ByCharacterIdProvider_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		ByCharacterIdProviderFunc: func(characterId uint32) model.Provider[inventory.Model] {
			return func() (inventory.Model, error) {
				return inventory.NewBuilder(characterId).Build(), nil
			}
		},
	}

	result, err := mockProcessor.ByCharacterIdProvider(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", result.CharacterId())
	}
}

func TestProcessorMock_ByCharacterIdProvider_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		ByCharacterIdProviderFunc: func(characterId uint32) model.Provider[inventory.Model] {
			return func() (inventory.Model, error) {
				return inventory.Model{}, errors.New("inventory not found")
			}
		},
	}

	_, err := mockProcessor.ByCharacterIdProvider(999)()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "inventory not found" {
		t.Errorf("Expected error message 'inventory not found', got '%s'", err.Error())
	}
}

func TestProcessorMock_GetByCharacterId_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetByCharacterIdFunc: func(characterId uint32) (inventory.Model, error) {
			return inventory.NewBuilder(characterId).Build(), nil
		},
	}

	result, err := mockProcessor.GetByCharacterId(123)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", result.CharacterId())
	}
}

func TestProcessorMock_GetByCharacterId_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetByCharacterIdFunc: func(characterId uint32) (inventory.Model, error) {
			return inventory.Model{}, errors.New("character not found")
		},
	}

	_, err := mockProcessor.GetByCharacterId(999)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default ByCharacterIdProvider returns model with characterId
	result, err := mockProcessor.ByCharacterIdProvider(123)()
	if err != nil {
		t.Errorf("Expected no error from default ByCharacterIdProvider, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected default CharacterId=123, got %d", result.CharacterId())
	}

	// Test default GetByCharacterId returns model with characterId
	result, err = mockProcessor.GetByCharacterId(456)
	if err != nil {
		t.Errorf("Expected no error from default GetByCharacterId, got %v", err)
	}

	if result.CharacterId() != 456 {
		t.Errorf("Expected default CharacterId=456, got %d", result.CharacterId())
	}
}
