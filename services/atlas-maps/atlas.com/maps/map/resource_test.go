package _map

import (
	"testing"
)

func TestTransform(t *testing.T) {
	tests := []struct {
		name     string
		id       uint32
		expected string
	}{
		{"zero", 0, "0"},
		{"typical character id", 12345, "12345"},
		{"large id", 4294967295, "4294967295"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Transform(tt.id)
			if err != nil {
				t.Fatalf("Transform returned error: %v", err)
			}
			if result.Id != tt.expected {
				t.Errorf("Expected Id %s, got %s", tt.expected, result.Id)
			}
		})
	}
}

func TestRestModel_GetID(t *testing.T) {
	m := RestModel{Id: "12345"}
	if m.GetID() != "12345" {
		t.Errorf("Expected GetID to return '12345', got '%s'", m.GetID())
	}
}

func TestRestModel_GetID_Empty(t *testing.T) {
	m := RestModel{}
	if m.GetID() != "" {
		t.Errorf("Expected GetID to return empty string, got '%s'", m.GetID())
	}
}

func TestRestModel_GetName(t *testing.T) {
	m := RestModel{}
	if m.GetName() != "characters" {
		t.Errorf("Expected GetName to return 'characters', got '%s'", m.GetName())
	}
}

func TestRestModel_SetID(t *testing.T) {
	m := RestModel{}
	err := m.SetID("67890")
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}
	if m.Id != "67890" {
		t.Errorf("Expected Id to be '67890', got '%s'", m.Id)
	}
}

func TestRestModel_SetID_Overwrite(t *testing.T) {
	m := RestModel{Id: "original"}
	err := m.SetID("new_value")
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}
	if m.Id != "new_value" {
		t.Errorf("Expected Id to be 'new_value', got '%s'", m.Id)
	}
}

func TestRestModel_SetID_Empty(t *testing.T) {
	m := RestModel{Id: "original"}
	err := m.SetID("")
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}
	if m.Id != "" {
		t.Errorf("Expected Id to be empty, got '%s'", m.Id)
	}
}

func TestTransform_MultipleIds(t *testing.T) {
	// Test that Transform produces consistent results
	ids := []uint32{1, 100, 1000, 10000, 100000}

	for _, id := range ids {
		result1, err1 := Transform(id)
		result2, err2 := Transform(id)

		if err1 != nil || err2 != nil {
			t.Fatalf("Transform returned error for id %d", id)
		}

		if result1.Id != result2.Id {
			t.Errorf("Transform not consistent for id %d: got %s and %s", id, result1.Id, result2.Id)
		}
	}
}
