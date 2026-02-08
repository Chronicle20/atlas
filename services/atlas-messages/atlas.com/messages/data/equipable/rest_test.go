package equipable

import (
	"testing"
)

// TestRestModel_GetName tests the GetName method
func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	name := rm.GetName()

	expected := "statistics"
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

	if name != "statistics" {
		t.Errorf("Expected 'statistics', got '%s'", name)
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
			name:     "Standard equipment ID",
			id:       1302000,
			expected: "1302000",
		},
		{
			name:     "Zero ID",
			id:       0,
			expected: "0",
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
			name:        "Valid equipment ID",
			idStr:       "1302000",
			expectedId:  1302000,
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
		Id:             1302000,
		Strength:       10,
		Dexterity:      5,
		Intelligence:   0,
		Luck:           0,
		Hp:             100,
		Mp:             50,
		WeaponAttack:   50,
		MagicAttack:    0,
		WeaponDefense:  20,
		MagicDefense:   10,
		Accuracy:       5,
		Avoidability:   5,
		Speed:          0,
		Jump:           0,
		Slots:          7,
	}

	model, err := Extract(rm)

	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if model.Strength() != rm.Strength {
		t.Errorf("Strength mismatch: expected %d, got %d", rm.Strength, model.Strength())
	}
	if model.Dexterity() != rm.Dexterity {
		t.Errorf("Dexterity mismatch: expected %d, got %d", rm.Dexterity, model.Dexterity())
	}
	if model.Hp() != rm.Hp {
		t.Errorf("Hp mismatch: expected %d, got %d", rm.Hp, model.Hp())
	}
	if model.Mp() != rm.Mp {
		t.Errorf("Mp mismatch: expected %d, got %d", rm.Mp, model.Mp())
	}
	if model.WeaponAttack() != rm.WeaponAttack {
		t.Errorf("WeaponAttack mismatch: expected %d, got %d", rm.WeaponAttack, model.WeaponAttack())
	}
	if model.Slots() != rm.Slots {
		t.Errorf("Slots mismatch: expected %d, got %d", rm.Slots, model.Slots())
	}
}

// TestExtract_ZeroValues tests extraction with zero values
func TestExtract_ZeroValues(t *testing.T) {
	rm := RestModel{}

	model, err := Extract(rm)

	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if model.Strength() != 0 {
		t.Errorf("Expected Strength=0, got %d", model.Strength())
	}
	if model.Hp() != 0 {
		t.Errorf("Expected Hp=0, got %d", model.Hp())
	}
}

// TestRestModel_GetReferences tests the GetReferences method
func TestRestModel_GetReferences(t *testing.T) {
	rm := RestModel{}
	refs := rm.GetReferences()

	if len(refs) != 1 {
		t.Fatalf("Expected 1 reference, got %d", len(refs))
	}

	if refs[0].Type != "slots" {
		t.Errorf("Expected reference type 'slots', got '%s'", refs[0].Type)
	}
}
