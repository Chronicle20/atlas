package shops_test

import (
	"atlas-channel/npc/shops"
	"atlas-channel/npc/shops/commodities"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := shops.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	commodity := commodities.NewModelBuilder().
		SetId(uuid.New()).
		SetTemplateId(1000).
		SetMesoPrice(500).
		MustBuild()

	model, err := shops.NewModelBuilder().
		SetNpcId(9000001).
		SetCommodities([]commodities.Model{commodity}).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.NpcId() != 9000001 {
		t.Errorf("model.NpcId() = %d, want 9000001", model.NpcId())
	}
	if len(model.Commodities()) != 1 {
		t.Errorf("len(model.Commodities()) = %d, want 1", len(model.Commodities()))
	}
}

func TestBuild_MissingNpcId(t *testing.T) {
	_, err := shops.NewModelBuilder().
		SetCommodities([]commodities.Model{}).
		Build()

	if err != shops.ErrInvalidNpcId {
		t.Errorf("Build() error = %v, want ErrInvalidNpcId", err)
	}
}

func TestBuild_ZeroNpcId(t *testing.T) {
	_, err := shops.NewModelBuilder().
		SetNpcId(0).
		Build()

	if err != shops.ErrInvalidNpcId {
		t.Errorf("Build() error = %v, want ErrInvalidNpcId", err)
	}
}

func TestCloneModel(t *testing.T) {
	commodity1 := commodities.NewModelBuilder().
		SetId(uuid.New()).
		SetTemplateId(1000).
		MustBuild()

	commodity2 := commodities.NewModelBuilder().
		SetId(uuid.New()).
		SetTemplateId(2000).
		MustBuild()

	original, err := shops.NewModelBuilder().
		SetNpcId(9000001).
		SetCommodities([]commodities.Model{commodity1}).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := shops.CloneModel(original).
		SetCommodities([]commodities.Model{commodity1, commodity2}).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if len(original.Commodities()) != 1 {
		t.Errorf("len(original.Commodities()) = %d, want 1", len(original.Commodities()))
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.NpcId() != 9000001 {
		t.Errorf("cloned.NpcId() = %d, want 9000001", cloned.NpcId())
	}
	if len(cloned.Commodities()) != 2 {
		t.Errorf("len(cloned.Commodities()) = %d, want 2", len(cloned.Commodities()))
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := shops.NewModelBuilder().
		SetNpcId(9000001).
		MustBuild()

	if model.NpcId() != 9000001 {
		t.Errorf("model.NpcId() = %d, want 9000001", model.NpcId())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	shops.NewModelBuilder().
		MustBuild() // Missing NpcId, should panic
}

func TestBuilderFluentChaining(t *testing.T) {
	commodity := commodities.NewModelBuilder().
		SetId(uuid.New()).
		SetTemplateId(1000).
		MustBuild()

	model, err := shops.NewModelBuilder().
		SetNpcId(9000001).
		SetCommodities([]commodities.Model{commodity}).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.NpcId() != 9000001 {
		t.Errorf("model.NpcId() = %d, want 9000001", model.NpcId())
	}
}
