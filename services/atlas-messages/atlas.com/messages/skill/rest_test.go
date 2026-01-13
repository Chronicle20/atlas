package skill

import (
	"testing"
)

// TestRestModel_GetName tests the GetName method
func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	name := rm.GetName()

	expected := "skills"
	if name != expected {
		t.Errorf("Expected GetName() to return '%s', got '%s'", expected, name)
	}
}

// TestRestModel_GetName_ValueReceiver tests that GetName uses value receiver
func TestRestModel_GetName_ValueReceiver(t *testing.T) {
	// This test verifies that GetName can be called on a value (not pointer)
	// which is required by api2go
	rm := RestModel{Id: 12345}
	name := rm.GetName()

	if name != "skills" {
		t.Errorf("Expected 'skills', got '%s'", name)
	}
}

// TestRestModel_GetID tests the GetID method
func TestRestModel_GetID(t *testing.T) {
	testCases := []struct {
		name     string
		id       uint32
		expected string
	}{
		{
			name:     "Standard skill ID",
			id:       1001004,
			expected: "1001004",
		},
		{
			name:     "Zero ID",
			id:       0,
			expected: "0",
		},
		{
			name:     "Warrior skill",
			id:       1121002,
			expected: "1121002",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rm := RestModel{Id: tc.id}
			result := rm.GetID()

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestRestModel_SetID tests the SetID method
func TestRestModel_SetID(t *testing.T) {
	testCases := []struct {
		name        string
		idStr       string
		expectedId  uint32
		expectError bool
	}{
		{
			name:        "Valid skill ID",
			idStr:       "1001004",
			expectedId:  1001004,
			expectError: false,
		},
		{
			name:        "Invalid - not a number",
			idStr:       "abc",
			expectedId:  0,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rm := &RestModel{}
			err := rm.SetID(tc.idStr)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for ID '%s', got nil", tc.idStr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if rm.Id != tc.expectedId {
					t.Errorf("Expected Id=%d, got Id=%d", tc.expectedId, rm.Id)
				}
			}
		})
	}
}

// TestExtract tests the Extract function
func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:          1001004,
		Level:       10,
		MasterLevel: 20,
	}

	model, err := Extract(rm)

	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if model.Id() != rm.Id {
		t.Errorf("Id mismatch: expected %d, got %d", rm.Id, model.Id())
	}
	if model.Level() != rm.Level {
		t.Errorf("Level mismatch: expected %d, got %d", rm.Level, model.Level())
	}
	if model.MasterLevel() != rm.MasterLevel {
		t.Errorf("MasterLevel mismatch: expected %d, got %d", rm.MasterLevel, model.MasterLevel())
	}
}

// TestExtract_ZeroValues tests extraction with zero values
func TestExtract_ZeroValues(t *testing.T) {
	rm := RestModel{}

	model, err := Extract(rm)

	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if model.Id() != 0 {
		t.Errorf("Expected Id=0, got %d", model.Id())
	}
	if model.Level() != 0 {
		t.Errorf("Expected Level=0, got %d", model.Level())
	}
	if model.MasterLevel() != 0 {
		t.Errorf("Expected MasterLevel=0, got %d", model.MasterLevel())
	}
}
