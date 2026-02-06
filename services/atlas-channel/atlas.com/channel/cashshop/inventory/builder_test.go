package inventory_test

import (
	"atlas-channel/cashshop/inventory"
	"atlas-channel/cashshop/inventory/compartment"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := inventory.NewModelBuilder(100)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	explorerCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeExplorer, 50).MustBuild()
	cygnusCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeCygnus, 50).MustBuild()
	legendCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeLegend, 50).MustBuild()

	model, err := inventory.NewModelBuilder(100).
		SetExplorer(explorerCompartment).
		SetCygnus(cygnusCompartment).
		SetLegend(legendCompartment).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.AccountId() != 100 {
		t.Errorf("model.AccountId() = %d, want 100", model.AccountId())
	}
	if len(model.Compartments()) != 3 {
		t.Errorf("len(model.Compartments()) = %d, want 3", len(model.Compartments()))
	}
}

func TestBuild_MissingAccountId(t *testing.T) {
	_, err := inventory.NewModelBuilder(0).Build()

	if !errors.Is(err, inventory.ErrInvalidAccountId) {
		t.Errorf("Build() error = %v, want ErrInvalidAccountId", err)
	}
}

func TestCloneModel(t *testing.T) {
	explorerCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeExplorer, 50).MustBuild()

	original, err := inventory.NewModelBuilder(100).
		SetExplorer(explorerCompartment).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cygnusCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeCygnus, 60).MustBuild()

	cloned, err := inventory.CloneModel(original).
		SetCygnus(cygnusCompartment).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if len(original.Compartments()) != 1 {
		t.Errorf("len(original.Compartments()) = %d, want 1", len(original.Compartments()))
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.AccountId() != 100 {
		t.Errorf("cloned.AccountId() = %d, want 100", cloned.AccountId())
	}
	if len(cloned.Compartments()) != 2 {
		t.Errorf("len(cloned.Compartments()) = %d, want 2", len(cloned.Compartments()))
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := inventory.NewModelBuilder(100).MustBuild()

	if model.AccountId() != 100 {
		t.Errorf("model.AccountId() = %d, want 100", model.AccountId())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	inventory.NewModelBuilder(0).MustBuild() // Missing AccountId, should panic
}

func TestSetCompartment(t *testing.T) {
	explorerCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeExplorer, 50).MustBuild()

	model, err := inventory.NewModelBuilder(100).
		SetCompartment(explorerCompartment).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	explorer := model.Explorer()
	if explorer.Type() != compartment.TypeExplorer {
		t.Errorf("explorer.Type() = %v, want TypeExplorer", explorer.Type())
	}
}

func TestCompartmentByType(t *testing.T) {
	explorerCompartment := compartment.NewModelBuilder(uuid.New(), 100, compartment.TypeExplorer, 50).MustBuild()

	model, err := inventory.NewModelBuilder(100).
		SetExplorer(explorerCompartment).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	c := model.CompartmentByType(compartment.TypeExplorer)
	if c.Type() != compartment.TypeExplorer {
		t.Errorf("CompartmentByType(TypeExplorer).Type() = %v, want TypeExplorer", c.Type())
	}
}
