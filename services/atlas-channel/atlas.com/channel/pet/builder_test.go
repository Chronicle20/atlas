package pet_test

import (
	"atlas-channel/pet"
	"errors"
	"testing"
)

func TestNewModelBuilder(t *testing.T) {
	builder := pet.NewModelBuilder(1, 1234567890, 5000000, "Fluffy")
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	model, err := pet.NewModelBuilder(1, 1234567890, 5000000, "Fluffy").
		SetLevel(10).
		SetCloseness(1000).
		SetFullness(100).
		SetOwnerID(100).
		SetSlot(0).
		SetX(50).
		SetY(100).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.Name() != "Fluffy" {
		t.Errorf("model.Name() = %s, want Fluffy", model.Name())
	}
	if model.Level() != 10 {
		t.Errorf("model.Level() = %d, want 10", model.Level())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := pet.NewModelBuilder(0, 1234567890, 5000000, "Fluffy").
		Build()

	if !errors.Is(err, pet.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := pet.NewModelBuilder(1, 1234567890, 5000000, "Fluffy").MustBuild()

	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	pet.NewModelBuilder(0, 1234567890, 5000000, "Fluffy").MustBuild() // Zero ID, should panic
}

func TestCloneModel(t *testing.T) {
	original, _ := pet.NewModelBuilder(1, 1234567890, 5000000, "Fluffy").
		SetLevel(10).
		SetCloseness(1000).
		Build()

	cloned, err := pet.CloneModel(original).
		SetLevel(15).
		SetCloseness(1500).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Level() != 10 {
		t.Errorf("original.Level() = %d, want 10", original.Level())
	}

	// Cloned should have new values
	if cloned.Level() != 15 {
		t.Errorf("cloned.Level() = %d, want 15", cloned.Level())
	}
	if cloned.Closeness() != 1500 {
		t.Errorf("cloned.Closeness() = %d, want 1500", cloned.Closeness())
	}
	// But preserve other fields
	if cloned.Name() != "Fluffy" {
		t.Errorf("cloned.Name() = %s, want Fluffy", cloned.Name())
	}
}
