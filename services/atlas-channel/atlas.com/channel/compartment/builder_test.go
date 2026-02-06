package compartment_test

import (
	"atlas-channel/compartment"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	id := uuid.New()
	builder := compartment.NewModelBuilder(id, 1, inventory.TypeValueEquip, 24)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestNewBuilder_Alias(t *testing.T) {
	id := uuid.New()
	builder := compartment.NewBuilder(id, 1, inventory.TypeValueEquip, 24)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()

	model, err := compartment.NewModelBuilder(id, 100, inventory.TypeValueEquip, 24).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.CharacterId() != 100 {
		t.Errorf("model.CharacterId() = %d, want 100", model.CharacterId())
	}
	if model.Type() != inventory.TypeValueEquip {
		t.Errorf("model.Type() = %v, want %v", model.Type(), inventory.TypeValueEquip)
	}
	if model.Capacity() != 24 {
		t.Errorf("model.Capacity() = %d, want 24", model.Capacity())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := compartment.NewModelBuilder(uuid.Nil, 100, inventory.TypeValueEquip, 24).
		Build()

	if !errors.Is(err, compartment.ErrMissingId) {
		t.Errorf("Build() error = %v, want ErrMissingId", err)
	}
}

func TestSetCapacity(t *testing.T) {
	id := uuid.New()

	model, err := compartment.NewModelBuilder(id, 100, inventory.TypeValueEquip, 24).
		SetCapacity(48).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Capacity() != 48 {
		t.Errorf("model.Capacity() = %d, want 48", model.Capacity())
	}
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()

	original, err := compartment.NewModelBuilder(id, 100, inventory.TypeValueEquip, 24).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := compartment.CloneModel(original).
		SetCapacity(48).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Capacity() != 24 {
		t.Errorf("original.Capacity() = %d, want 24", original.Capacity())
	}

	// Cloned should have new capacity
	if cloned.Id() != id {
		t.Errorf("cloned.Id() = %v, want %v", cloned.Id(), id)
	}
	if cloned.Capacity() != 48 {
		t.Errorf("cloned.Capacity() = %d, want 48", cloned.Capacity())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	id := uuid.New()
	model := compartment.NewModelBuilder(id, 100, inventory.TypeValueEquip, 24).
		MustBuild()

	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	compartment.NewModelBuilder(uuid.Nil, 100, inventory.TypeValueEquip, 24).
		MustBuild() // Missing ID, should panic
}
