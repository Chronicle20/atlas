package chalkboard

import (
	"testing"
)

func TestTransform(t *testing.T) {
	m := NewBuilder(12345).SetMessage("Hello, World!").Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != 12345 {
		t.Errorf("Id mismatch. Expected 12345, got %v", rm.Id)
	}

	if rm.Message != "Hello, World!" {
		t.Errorf("Message mismatch. Expected 'Hello, World!', got %q", rm.Message)
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != "chalkboards" {
		t.Errorf("GetName mismatch. Expected 'chalkboards', got '%v'", rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	rm := RestModel{Id: 789}
	if rm.GetID() != "789" {
		t.Errorf("GetID mismatch. Expected '789', got '%v'", rm.GetID())
	}
}

func TestRestModelGetIDZero(t *testing.T) {
	rm := RestModel{Id: 0}
	if rm.GetID() != "0" {
		t.Errorf("GetID mismatch. Expected '0', got '%v'", rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("321")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}

	if rm.Id != 321 {
		t.Errorf("Id mismatch after SetID. Expected 321, got %v", rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("notanumber")
	if err == nil {
		t.Fatal("Expected error for invalid id, got nil")
	}
}

func TestRestModelSetIDNegative(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("-1")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	// Note: strconv.Atoi allows negative numbers, they wrap around for uint32
}

func TestTransformEmptyMessage(t *testing.T) {
	m := NewBuilder(1).Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Message != "" {
		t.Errorf("Expected empty message, got %q", rm.Message)
	}
}
