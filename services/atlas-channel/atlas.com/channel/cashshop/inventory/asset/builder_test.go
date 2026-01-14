package asset_test

import (
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/item"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	id := uuid.New()
	compartmentId := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()

	builder := asset.NewModelBuilder(id, compartmentId, testItem)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()
	compartmentId := uuid.New()
	testItem := item.NewModelBuilder().
		SetId(1).
		SetTemplateId(5000000).
		SetQuantity(1).
		MustBuild()

	model, err := asset.NewModelBuilder(id, compartmentId, testItem).Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.CompartmentId() != compartmentId {
		t.Errorf("model.CompartmentId() = %v, want %v", model.CompartmentId(), compartmentId)
	}
	if model.TemplateId() != 5000000 {
		t.Errorf("model.TemplateId() = %d, want 5000000", model.TemplateId())
	}
}

func TestBuild_MissingId(t *testing.T) {
	compartmentId := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()

	_, err := asset.NewModelBuilder(uuid.Nil, compartmentId, testItem).Build()

	if err != asset.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestBuild_MissingCompartmentId(t *testing.T) {
	id := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()

	_, err := asset.NewModelBuilder(id, uuid.Nil, testItem).Build()

	if err != asset.ErrInvalidCompartmentId {
		t.Errorf("Build() error = %v, want ErrInvalidCompartmentId", err)
	}
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()
	compartmentId := uuid.New()
	testItem1 := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()
	testItem2 := item.NewModelBuilder().SetId(2).SetTemplateId(5000001).MustBuild()

	original, err := asset.NewModelBuilder(id, compartmentId, testItem1).Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := asset.CloneModel(original).
		SetItem(testItem2).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.TemplateId() != 5000000 {
		t.Errorf("original.TemplateId() = %d, want 5000000", original.TemplateId())
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.Id() != id {
		t.Errorf("cloned.Id() = %v, want %v", cloned.Id(), id)
	}
	if cloned.CompartmentId() != compartmentId {
		t.Errorf("cloned.CompartmentId() = %v, want %v", cloned.CompartmentId(), compartmentId)
	}
	if cloned.TemplateId() != 5000001 {
		t.Errorf("cloned.TemplateId() = %d, want 5000001", cloned.TemplateId())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	id := uuid.New()
	compartmentId := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()

	model := asset.NewModelBuilder(id, compartmentId, testItem).MustBuild()

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

	compartmentId := uuid.New()
	testItem := item.NewModelBuilder().SetId(1).SetTemplateId(5000000).MustBuild()

	asset.NewModelBuilder(uuid.Nil, compartmentId, testItem).MustBuild() // Missing ID, should panic
}
