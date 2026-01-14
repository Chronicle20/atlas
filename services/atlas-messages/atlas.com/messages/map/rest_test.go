package _map

import (
	"testing"
)

// TestRestModel_GetName tests the GetName method
func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	name := rm.GetName()

	expected := "characters"
	if name != expected {
		t.Errorf("Expected GetName() to return '%s', got '%s'", expected, name)
	}
}

// TestRestModel_GetName_ValueReceiver tests that GetName uses value receiver
func TestRestModel_GetName_ValueReceiver(t *testing.T) {
	// This test verifies that GetName can be called on a value (not pointer)
	// which is required by api2go
	rm := RestModel{Id: "12345"}
	name := rm.GetName()

	if name != "characters" {
		t.Errorf("Expected 'characters', got '%s'", name)
	}
}

// TestRestModel_GetID tests the GetID method
func TestRestModel_GetID(t *testing.T) {
	testCases := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "Standard ID",
			id:       "12345",
			expected: "12345",
		},
		{
			name:     "Zero ID",
			id:       "0",
			expected: "0",
		},
		{
			name:     "Empty ID",
			id:       "",
			expected: "",
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
		name       string
		idStr      string
		expectedId string
	}{
		{
			name:       "Valid ID",
			idStr:      "12345",
			expectedId: "12345",
		},
		{
			name:       "Any string",
			idStr:      "abc",
			expectedId: "abc",
		},
		{
			name:       "Empty string",
			idStr:      "",
			expectedId: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rm := &RestModel{}
			err := rm.SetID(tc.idStr)

			// SetID in map/rest.go never returns an error
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if rm.Id != tc.expectedId {
				t.Errorf("Expected Id='%s', got Id='%s'", tc.expectedId, rm.Id)
			}
		})
	}
}

// TestExtract tests the Extract function
func TestExtract(t *testing.T) {
	testCases := []struct {
		name        string
		id          string
		expected    uint32
		expectError bool
	}{
		{
			name:        "Valid numeric ID",
			id:          "12345",
			expected:    12345,
			expectError: false,
		},
		{
			name:        "Zero ID",
			id:          "0",
			expected:    0,
			expectError: false,
		},
		{
			name:        "Invalid - not a number",
			id:          "abc",
			expected:    0,
			expectError: true,
		},
		{
			name:        "Empty string",
			id:          "",
			expected:    0,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rm := RestModel{Id: tc.id}
			result, err := Extract(rm)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for ID '%s', got nil", tc.id)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tc.expected {
					t.Errorf("Expected %d, got %d", tc.expected, result)
				}
			}
		})
	}
}
