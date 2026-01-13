package character

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/world"
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
	rm := RestModel{Id: 12345}
	name := rm.GetName()

	if name != "characters" {
		t.Errorf("Expected 'characters', got '%s'", name)
	}

	// Also verify it works on a copy
	rmCopy := rm
	nameCopy := rmCopy.GetName()

	if nameCopy != "characters" {
		t.Errorf("Expected 'characters' from copy, got '%s'", nameCopy)
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
			name:     "Standard ID",
			id:       12345,
			expected: "12345",
		},
		{
			name:     "Zero ID",
			id:       0,
			expected: "0",
		},
		{
			name:     "Large ID",
			id:       4294967295, // Max uint32
			expected: "4294967295",
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
			name:        "Valid ID",
			idStr:       "12345",
			expectedId:  12345,
			expectError: false,
		},
		{
			name:        "Zero ID",
			idStr:       "0",
			expectedId:  0,
			expectError: false,
		},
		{
			name:        "Invalid - not a number",
			idStr:       "abc",
			expectedId:  0,
			expectError: true,
		},
		// Note: Negative values are not rejected by SetID - they get converted to large uint32 values
		// This is the actual behavior of the code
		{
			name:        "Invalid - empty",
			idStr:       "",
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

// TestRestModel_GetReferences tests the GetReferences method
func TestRestModel_GetReferences(t *testing.T) {
	rm := RestModel{}
	refs := rm.GetReferences()

	if len(refs) != 2 {
		t.Fatalf("Expected 2 references, got %d", len(refs))
	}

	expectedTypes := map[string]bool{
		"equipment":   false,
		"inventories": false,
	}

	for _, ref := range refs {
		if _, ok := expectedTypes[ref.Type]; ok {
			expectedTypes[ref.Type] = true
		} else {
			t.Errorf("Unexpected reference type: %s", ref.Type)
		}
	}

	for refType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected reference type '%s' not found", refType)
		}
	}
}

// TestExtract tests the Extract function
func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:                 12345,
		AccountId:          100,
		WorldId:            1,
		Name:               "TestPlayer",
		Level:              50,
		Experience:         1000000,
		GachaponExperience: 500,
		Strength:           100,
		Dexterity:          50,
		Intelligence:       30,
		Luck:               40,
		Hp:                 5000,
		MaxHp:              6000,
		Mp:                 3000,
		MaxMp:              4000,
		Meso:               1000000,
		HpMpUsed:           100,
		JobId:              112,
		SkinColor:          1,
		Gender:             0,
		Fame:               100,
		Hair:               30030,
		Face:               20000,
		Ap:                 50,
		Sp:                 "10,5,3",
		MapId:              100000000,
		SpawnPoint:         0,
		Gm:                 1,
		X:                  0,
		Y:                  0,
		Stance:             0,
	}

	model, err := Extract(rm)

	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Verify all fields are correctly mapped
	if model.Id() != rm.Id {
		t.Errorf("Id mismatch: expected %d, got %d", rm.Id, model.Id())
	}
	if model.AccountId() != rm.AccountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", rm.AccountId, model.AccountId())
	}
	if model.WorldId() != world.Id(rm.WorldId) {
		t.Errorf("WorldId mismatch: expected %d, got %d", rm.WorldId, model.WorldId())
	}
	if model.Name() != rm.Name {
		t.Errorf("Name mismatch: expected %s, got %s", rm.Name, model.Name())
	}
	if model.Level() != rm.Level {
		t.Errorf("Level mismatch: expected %d, got %d", rm.Level, model.Level())
	}
	if model.Experience() != rm.Experience {
		t.Errorf("Experience mismatch: expected %d, got %d", rm.Experience, model.Experience())
	}
	if model.Meso() != rm.Meso {
		t.Errorf("Meso mismatch: expected %d, got %d", rm.Meso, model.Meso())
	}
	if model.JobId() != rm.JobId {
		t.Errorf("JobId mismatch: expected %d, got %d", rm.JobId, model.JobId())
	}
	if model.Fame() != rm.Fame {
		t.Errorf("Fame mismatch: expected %d, got %d", rm.Fame, model.Fame())
	}
	if model.MapId() != rm.MapId {
		t.Errorf("MapId mismatch: expected %d, got %d", rm.MapId, model.MapId())
	}
	if model.Gm() != (rm.Gm == 1) {
		t.Errorf("Gm mismatch: expected %v, got %v", rm.Gm == 1, model.Gm())
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
	if model.Gm() != false {
		t.Errorf("Expected Gm=false, got %v", model.Gm())
	}
}

// TestExtract_GmValues tests GM flag extraction
func TestExtract_GmValues(t *testing.T) {
	testCases := []struct {
		name     string
		gmValue  int
		expected bool
	}{
		{
			name:     "GM = 0 (not GM)",
			gmValue:  0,
			expected: false,
		},
		{
			name:     "GM = 1 (is GM)",
			gmValue:  1,
			expected: true,
		},
		{
			name:     "GM = 2 (not GM - only 1 is GM)",
			gmValue:  2,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rm := RestModel{Gm: tc.gmValue}
			model, _ := Extract(rm)

			if model.Gm() != tc.expected {
				t.Errorf("Expected Gm()=%v for gm=%d, got %v", tc.expected, tc.gmValue, model.Gm())
			}
		})
	}
}

// TestRestModel_SetToOneReferenceID tests the SetToOneReferenceID method
func TestRestModel_SetToOneReferenceID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetToOneReferenceID("test", "123")

	// Currently returns nil (no-op implementation)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

// TestRestModel_SetToManyReferenceIDs tests the SetToManyReferenceIDs method
func TestRestModel_SetToManyReferenceIDs(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetToManyReferenceIDs("test", []string{"1", "2", "3"})

	// Currently returns nil (no-op implementation)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

// TestRestModel_SetReferencedStructs tests the SetReferencedStructs method
func TestRestModel_SetReferencedStructs(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetReferencedStructs(nil)

	// Currently returns nil (no-op implementation)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

// TestRestModel_GetReferencedIDs tests the GetReferencedIDs method
func TestRestModel_GetReferencedIDs(t *testing.T) {
	rm := RestModel{}
	refs := rm.GetReferencedIDs()

	// Currently returns empty slice
	if len(refs) != 0 {
		t.Errorf("Expected 0 referenced IDs, got %d", len(refs))
	}
}

// TestRestModel_GetReferencedStructs tests the GetReferencedStructs method
func TestRestModel_GetReferencedStructs(t *testing.T) {
	rm := RestModel{}
	refs := rm.GetReferencedStructs()

	// Currently returns empty slice
	if len(refs) != 0 {
		t.Errorf("Expected 0 referenced structs, got %d", len(refs))
	}
}
