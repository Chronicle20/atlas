package purchaserecord

import (
	"errors"
	"testing"
)

func TestNewModelBuilder(t *testing.T) {
	builder := NewModelBuilder(5555)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	m, err := NewModelBuilder(5555).SetPurchased(true).SetCount(2).Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if m.SerialNumber() != 5555 {
		t.Errorf("m.SerialNumber() = %d, want 5555", m.SerialNumber())
	}
	if !m.Purchased() {
		t.Error("m.Purchased() = false, want true")
	}
	if m.Count() != 2 {
		t.Errorf("m.Count() = %d, want 2", m.Count())
	}
}

func TestBuild_MissingSerialNumber(t *testing.T) {
	_, err := NewModelBuilder(0).Build()
	if !errors.Is(err, ErrInvalidSerialNumber) {
		t.Errorf("Build() error = %v, want ErrInvalidSerialNumber", err)
	}
}

func TestCloneModel(t *testing.T) {
	original, err := NewModelBuilder(5555).SetPurchased(true).SetCount(2).Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := CloneModel(original).SetCount(3).Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Count() != 2 {
		t.Errorf("original.Count() = %d, want 2", original.Count())
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.SerialNumber() != 5555 {
		t.Errorf("cloned.SerialNumber() = %d, want 5555", cloned.SerialNumber())
	}
	if !cloned.Purchased() {
		t.Error("cloned.Purchased() = false, want true")
	}
	if cloned.Count() != 3 {
		t.Errorf("cloned.Count() = %d, want 3", cloned.Count())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	m := NewModelBuilder(5555).MustBuild()
	if m.SerialNumber() != 5555 {
		t.Errorf("m.SerialNumber() = %d, want 5555", m.SerialNumber())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	NewModelBuilder(0).MustBuild() // Missing serial number, should panic
}
