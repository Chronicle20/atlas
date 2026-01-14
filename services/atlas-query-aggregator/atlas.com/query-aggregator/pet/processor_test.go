package pet_test

import (
	"atlas-query-aggregator/pet"
	"atlas-query-aggregator/pet/mock"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
)

func TestProcessorMock_GetPets_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPetsFunc: func(characterId uint32) model.Provider[[]pet.Model] {
			return func() ([]pet.Model, error) {
				return []pet.Model{
					pet.NewModel(1001, 0),  // Spawned
					pet.NewModel(1002, 1),  // Spawned
					pet.NewModel(1003, -1), // Not spawned
				}, nil
			}
		},
	}

	pets, err := mockProcessor.GetPets(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(pets) != 3 {
		t.Errorf("Expected 3 pets, got %d", len(pets))
	}

	// Verify spawned status
	if !pets[0].IsSpawned() {
		t.Error("Expected pet 1001 to be spawned")
	}

	if !pets[1].IsSpawned() {
		t.Error("Expected pet 1002 to be spawned")
	}

	if pets[2].IsSpawned() {
		t.Error("Expected pet 1003 to not be spawned")
	}
}

func TestProcessorMock_GetPets_Empty(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPetsFunc: func(characterId uint32) model.Provider[[]pet.Model] {
			return func() ([]pet.Model, error) {
				return []pet.Model{}, nil
			}
		},
	}

	pets, err := mockProcessor.GetPets(456)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(pets) != 0 {
		t.Errorf("Expected 0 pets, got %d", len(pets))
	}
}

func TestProcessorMock_GetPets_Error(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetPetsFunc: func(characterId uint32) model.Provider[[]pet.Model] {
			return func() ([]pet.Model, error) {
				return nil, errors.New("pet service unavailable")
			}
		},
	}

	_, err := mockProcessor.GetPets(123)()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_GetSpawnedPetCount_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSpawnedPetCountFunc: func(characterId uint32) model.Provider[int] {
			return func() (int, error) {
				return 2, nil
			}
		},
	}

	count, err := mockProcessor.GetSpawnedPetCount(123)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count=2, got %d", count)
	}
}

func TestProcessorMock_GetSpawnedPetCount_Zero(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSpawnedPetCountFunc: func(characterId uint32) model.Provider[int] {
			return func() (int, error) {
				return 0, nil
			}
		},
	}

	count, err := mockProcessor.GetSpawnedPetCount(456)()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count=0, got %d", count)
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetPets returns empty slice
	pets, err := mockProcessor.GetPets(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetPets, got %v", err)
	}

	if len(pets) != 0 {
		t.Errorf("Expected default pets count=0, got %d", len(pets))
	}

	// Test default GetSpawnedPetCount returns 0
	count, err := mockProcessor.GetSpawnedPetCount(123)()
	if err != nil {
		t.Errorf("Expected no error from default GetSpawnedPetCount, got %v", err)
	}

	if count != 0 {
		t.Errorf("Expected default count=0, got %d", count)
	}
}

func TestModel_IsSpawned(t *testing.T) {
	tests := []struct {
		name     string
		slot     int8
		expected bool
	}{
		{"slot 0 is spawned", 0, true},
		{"slot 1 is spawned", 1, true},
		{"slot 2 is spawned", 2, true},
		{"slot -1 is not spawned", -1, false},
		{"slot -50 is not spawned", -50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := pet.NewModel(1, tt.slot)
			if p.IsSpawned() != tt.expected {
				t.Errorf("Expected IsSpawned()=%v for slot=%d, got %v", tt.expected, tt.slot, p.IsSpawned())
			}
		})
	}
}
