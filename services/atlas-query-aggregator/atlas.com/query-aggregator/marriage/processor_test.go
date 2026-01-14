package marriage_test

import (
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/marriage/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

func TestProcessorMock_GetMarriageGifts_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetMarriageGiftsFunc: func(characterId uint32) model.Provider[marriage.Model] {
			return func() (marriage.Model, error) {
				return marriage.NewModelBuilder().
					SetCharacterId(characterId).
					SetHasUnclaimedGifts(true).
					Build(), nil
			}
		},
	}

	result, err := mockProcessor.GetMarriageGifts(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.CharacterId() != 123 {
		t.Errorf("Expected CharacterId=123, got %d", result.CharacterId())
	}

	if !result.HasUnclaimedGifts() {
		t.Error("Expected HasUnclaimedGifts=true")
	}
}

func TestProcessorMock_GetMarriageGifts_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetMarriageGiftsFunc: func(characterId uint32) model.Provider[marriage.Model] {
			return func() (marriage.Model, error) {
				return marriage.Model{}, errors.New("marriage service unavailable")
			}
		},
	}

	_, err := mockProcessor.GetMarriageGifts(123)()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "marriage service unavailable" {
		t.Errorf("Expected error message 'marriage service unavailable', got '%s'", err.Error())
	}
}

func TestProcessorMock_HasUnclaimedGifts_True(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		HasUnclaimedGiftsFunc: func(characterId uint32) model.Provider[bool] {
			return func() (bool, error) {
				return true, nil
			}
		},
	}

	hasGifts, err := mockProcessor.HasUnclaimedGifts(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !hasGifts {
		t.Error("Expected HasUnclaimedGifts=true")
	}
}

func TestProcessorMock_HasUnclaimedGifts_False(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		HasUnclaimedGiftsFunc: func(characterId uint32) model.Provider[bool] {
			return func() (bool, error) {
				return false, nil
			}
		},
	}

	hasGifts, err := mockProcessor.HasUnclaimedGifts(456)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if hasGifts {
		t.Error("Expected HasUnclaimedGifts=false")
	}
}

func TestProcessorMock_HasUnclaimedGifts_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		HasUnclaimedGiftsFunc: func(characterId uint32) model.Provider[bool] {
			return func() (bool, error) {
				return false, errors.New("service unavailable")
			}
		},
	}

	_, err := mockProcessor.HasUnclaimedGifts(123)()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_GetUnclaimedGiftCount_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetUnclaimedGiftCountFunc: func(characterId uint32) model.Provider[int] {
			return func() (int, error) {
				return 5, nil
			}
		},
	}

	count, err := mockProcessor.GetUnclaimedGiftCount(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 5 {
		t.Errorf("Expected count=5, got %d", count)
	}
}

func TestProcessorMock_GetUnclaimedGiftCount_Zero(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetUnclaimedGiftCountFunc: func(characterId uint32) model.Provider[int] {
			return func() (int, error) {
				return 0, nil
			}
		},
	}

	count, err := mockProcessor.GetUnclaimedGiftCount(456)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count=0, got %d", count)
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetMarriageGifts returns model with hasUnclaimedGifts=false
	result, err := mockProcessor.GetMarriageGifts(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetMarriageGifts, got %v", err)
	}

	if result.HasUnclaimedGifts() {
		t.Error("Expected default HasUnclaimedGifts=false")
	}

	// Test default HasUnclaimedGifts returns false
	hasGifts, err := mockProcessor.HasUnclaimedGifts(123)()
	if err != nil {
		t.Errorf("Expected no error from default HasUnclaimedGifts, got %v", err)
	}

	if hasGifts {
		t.Error("Expected default HasUnclaimedGifts=false")
	}

	// Test default GetUnclaimedGiftCount returns 0
	count, err := mockProcessor.GetUnclaimedGiftCount(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetUnclaimedGiftCount, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected default count=0, got %d", count)
	}
}
