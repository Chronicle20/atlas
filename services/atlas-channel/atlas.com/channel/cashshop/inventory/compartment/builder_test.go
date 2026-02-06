package compartment_test

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/cashshop/item"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	id := uuid.New()
	builder := compartment.NewModelBuilder(id, 100, compartment.TypeExplorer, 50)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()
	testAsset := asset.NewModelBuilder(uuid.New(), id, testItem).MustBuild()

	model, err := compartment.NewModelBuilder(id, 100, compartment.TypeExplorer, 50).
		AddAsset(testAsset).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.AccountId() != 100 {
		t.Errorf("model.AccountId() = %d, want 100", model.AccountId())
	}
	if model.Type() != compartment.TypeExplorer {
		t.Errorf("model.Type() = %v, want TypeExplorer", model.Type())
	}
	if model.Capacity() != 50 {
		t.Errorf("model.Capacity() = %d, want 50", model.Capacity())
	}
	if len(model.Assets()) != 1 {
		t.Errorf("len(model.Assets()) = %d, want 1", len(model.Assets()))
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := compartment.NewModelBuilder(uuid.Nil, 100, compartment.TypeExplorer, 50).Build()

	if !errors.Is(err, compartment.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestBuild_MissingAccountId(t *testing.T) {
	id := uuid.New()
	_, err := compartment.NewModelBuilder(id, 0, compartment.TypeExplorer, 50).Build()

	if !errors.Is(err, compartment.ErrInvalidAccountId) {
		t.Errorf("Build() error = %v, want ErrInvalidAccountId", err)
	}
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()
	testAsset := asset.NewModelBuilder(uuid.New(), id, testItem).MustBuild()

	original, err := compartment.NewModelBuilder(id, 100, compartment.TypeExplorer, 50).
		AddAsset(testAsset).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := compartment.CloneModel(original).
		SetCapacity(100).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Capacity() != 50 {
		t.Errorf("original.Capacity() = %d, want 50", original.Capacity())
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.Id() != id {
		t.Errorf("cloned.Id() = %v, want %v", cloned.Id(), id)
	}
	if cloned.AccountId() != 100 {
		t.Errorf("cloned.AccountId() = %d, want 100", cloned.AccountId())
	}
	if cloned.Capacity() != 100 {
		t.Errorf("cloned.Capacity() = %d, want 100", cloned.Capacity())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	id := uuid.New()
	model := compartment.NewModelBuilder(id, 100, compartment.TypeExplorer, 50).MustBuild()

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

	compartment.NewModelBuilder(uuid.Nil, 100, compartment.TypeExplorer, 50).MustBuild() // Missing ID, should panic
}

func TestAddAsset(t *testing.T) {
	id := uuid.New()
	testItem1 := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()
	testAsset1 := asset.NewModelBuilder(uuid.New(), id, testItem1).MustBuild()
	testItem2 := item.NewModelBuilder().SetId(2).SetTemplateId(5000001).MustBuild()
	testAsset2 := asset.NewModelBuilder(uuid.New(), id, testItem2).MustBuild()

	model, err := compartment.NewModelBuilder(id, 100, compartment.TypeExplorer, 50).
		AddAsset(testAsset1).
		AddAsset(testAsset2).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(model.Assets()) != 2 {
		t.Errorf("len(model.Assets()) = %d, want 2", len(model.Assets()))
	}
}

func TestSetAssets(t *testing.T) {
	id := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()
	testAsset := asset.NewModelBuilder(uuid.New(), id, testItem).MustBuild()

	model, err := compartment.NewModelBuilder(id, 100, compartment.TypeExplorer, 50).
		SetAssets([]asset.Model{testAsset}).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(model.Assets()) != 1 {
		t.Errorf("len(model.Assets()) = %d, want 1", len(model.Assets()))
	}
}
