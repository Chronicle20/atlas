package compartment_test

import (
	"atlas-query-aggregator/compartment"
	"atlas-query-aggregator/compartment/mock"
	"errors"
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

func TestProcessorMock_ByCharacterIdAndTypeProvider_Success(t *testing.T) {
	testId := uuid.New()
	mockProcessor := &mock.ProcessorImpl{
		ByCharacterIdAndTypeProviderFunc: func(characterId uint32, inventoryType inventory.Type) model.Provider[compartment.Model] {
			return func() (compartment.Model, error) {
				return compartment.NewBuilder(testId, characterId, inventoryType, 100).Build(), nil
			}
		},
	}

	result, err := mockProcessor.ByCharacterIdAndTypeProvider(123, inventory.TypeValueUse)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", result.CharacterId())
	}

	if result.Type() != inventory.TypeValueUse {
		t.Errorf("Expected Type=TypeValueUse, got %v", result.Type())
	}

	if result.Capacity() != 100 {
		t.Errorf("Expected Capacity=100, got %d", result.Capacity())
	}
}

func TestProcessorMock_ByCharacterIdAndTypeProvider_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		ByCharacterIdAndTypeProviderFunc: func(characterId uint32, inventoryType inventory.Type) model.Provider[compartment.Model] {
			return func() (compartment.Model, error) {
				return compartment.Model{}, errors.New("compartment not found")
			}
		},
	}

	_, err := mockProcessor.ByCharacterIdAndTypeProvider(999, inventory.TypeValueUse)()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "compartment not found" {
		t.Errorf("Expected error message 'compartment not found', got '%s'", err.Error())
	}
}

func TestProcessorMock_GetByType_Success(t *testing.T) {
	testId := uuid.New()
	mockProcessor := &mock.ProcessorImpl{
		GetByTypeFunc: func(characterId uint32, inventoryType inventory.Type) (compartment.Model, error) {
			return compartment.NewBuilder(testId, characterId, inventoryType, 50).Build(), nil
		},
	}

	result, err := mockProcessor.GetByType(123, inventory.TypeValueEquip)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", result.CharacterId())
	}

	if result.Type() != inventory.TypeValueEquip {
		t.Errorf("Expected Type=TypeValueEquip, got %v", result.Type())
	}
}

func TestProcessorMock_GetByType_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetByTypeFunc: func(characterId uint32, inventoryType inventory.Type) (compartment.Model, error) {
			return compartment.Model{}, errors.New("service unavailable")
		},
	}

	_, err := mockProcessor.GetByType(123, inventory.TypeValueUse)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default ByCharacterIdAndTypeProvider returns model with default values
	result, err := mockProcessor.ByCharacterIdAndTypeProvider(123, inventory.TypeValueUse)()
	if err != nil {
		t.Errorf("Expected no error from default ByCharacterIdAndTypeProvider, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected default CharacterId=123, got %d", result.CharacterId())
	}

	if result.Type() != inventory.TypeValueUse {
		t.Errorf("Expected default Type=TypeValueUse, got %v", result.Type())
	}

	if result.Capacity() != 100 {
		t.Errorf("Expected default Capacity=100, got %d", result.Capacity())
	}

	// Test default GetByType returns model with default values
	result, err = mockProcessor.GetByType(456, inventory.TypeValueETC)
	if err != nil {
		t.Errorf("Expected no error from default GetByType, got %v", err)
	}

	if result.CharacterId() != 456 {
		t.Errorf("Expected default CharacterId=456, got %d", result.CharacterId())
	}
}

func TestProcessorMock_AllInventoryTypes(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	inventoryTypes := []inventory.Type{
		inventory.TypeValueEquip,
		inventory.TypeValueUse,
		inventory.TypeValueSetup,
		inventory.TypeValueETC,
		inventory.TypeValueCash,
	}

	for _, invType := range inventoryTypes {
		t.Run(fmt.Sprintf("type_%d", invType), func(t *testing.T) {
			result, err := mockProcessor.GetByType(123, invType)
			if err != nil {
				t.Errorf("Expected no error for %v, got %v", invType, err)
			}

			if result.Type() != invType {
				t.Errorf("Expected Type=%v, got %v", invType, result.Type())
			}
		})
	}
}
