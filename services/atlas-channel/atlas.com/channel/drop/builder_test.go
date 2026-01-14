package drop_test

import (
	"atlas-channel/drop"
	"testing"
)

func TestNewModelBuilder(t *testing.T) {
	builder := drop.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	model, err := drop.NewModelBuilder().
		SetId(1).
		SetItem(1000, 10).
		SetMeso(100).
		SetType(1).
		SetPosition(100, 200).
		SetOwner(1001, 0).
		SetDropper(2001, 50, 150).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.ItemId() != 1000 {
		t.Errorf("model.ItemId() = %d, want 1000", model.ItemId())
	}
	if model.Quantity() != 10 {
		t.Errorf("model.Quantity() = %d, want 10", model.Quantity())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := drop.NewModelBuilder().
		SetItem(1000, 10).
		Build()

	if err != drop.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := drop.NewModelBuilder().
		SetId(1).
		MustBuild()

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

	drop.NewModelBuilder().MustBuild() // Missing ID, should panic
}

func TestCloneModel(t *testing.T) {
	original, _ := drop.NewModelBuilder().
		SetId(1).
		SetItem(1000, 10).
		SetPosition(100, 200).
		Build()

	cloned, err := drop.CloneModel(original).
		SetPosition(300, 400).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.X() != 100 {
		t.Errorf("original.X() = %d, want 100", original.X())
	}

	// Cloned should have new position
	if cloned.X() != 300 {
		t.Errorf("cloned.X() = %d, want 300", cloned.X())
	}
	if cloned.ItemId() != 1000 {
		t.Errorf("cloned.ItemId() = %d, want 1000", cloned.ItemId())
	}
}
