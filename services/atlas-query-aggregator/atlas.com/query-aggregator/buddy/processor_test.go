package buddy_test

import (
	"atlas-query-aggregator/buddy"
	"atlas-query-aggregator/buddy/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

func TestProcessorMock_GetBuddyList_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetBuddyListFunc: func(characterId uint32) model.Provider[buddy.Model] {
			return func() (buddy.Model, error) {
				return buddy.NewModel(characterId, 50), nil
			}
		},
	}

	result, err := mockProcessor.GetBuddyList(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", result.CharacterId())
	}

	if result.Capacity() != 50 {
		t.Errorf("Expected Capacity=50, got %d", result.Capacity())
	}
}

func TestProcessorMock_GetBuddyList_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetBuddyListFunc: func(characterId uint32) model.Provider[buddy.Model] {
			return func() (buddy.Model, error) {
				return buddy.Model{}, errors.New("buddy service unavailable")
			}
		},
	}

	_, err := mockProcessor.GetBuddyList(123)()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "buddy service unavailable" {
		t.Errorf("Expected error message 'buddy service unavailable', got '%s'", err.Error())
	}
}

func TestProcessorMock_GetBuddyCapacity_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetBuddyCapacityFunc: func(characterId uint32) model.Provider[byte] {
			return func() (byte, error) {
				return 100, nil
			}
		},
	}

	capacity, err := mockProcessor.GetBuddyCapacity(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if capacity != 100 {
		t.Errorf("Expected capacity=100, got %d", capacity)
	}
}

func TestProcessorMock_GetBuddyCapacity_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetBuddyCapacityFunc: func(characterId uint32) model.Provider[byte] {
			return func() (byte, error) {
				return 0, errors.New("service unavailable")
			}
		},
	}

	_, err := mockProcessor.GetBuddyCapacity(123)()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetBuddyList returns model with default capacity
	result, err := mockProcessor.GetBuddyList(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetBuddyList, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected default CharacterId=123, got %d", result.CharacterId())
	}

	if result.Capacity() != 20 {
		t.Errorf("Expected default Capacity=20, got %d", result.Capacity())
	}

	// Test default GetBuddyCapacity returns 20
	capacity, err := mockProcessor.GetBuddyCapacity(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetBuddyCapacity, got %v", err)
	}

	if capacity != 20 {
		t.Errorf("Expected default capacity=20, got %d", capacity)
	}
}
