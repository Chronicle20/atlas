package commodities_test

import (
	"atlas-channel/npc/shops/commodities"
	"testing"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := commodities.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	id := uuid.New()
	model, err := commodities.NewModelBuilder().
		SetId(id).
		SetTemplateId(1000).
		SetMesoPrice(500).
		SetDiscountRate(10).
		SetTokenTemplateId(2000).
		SetTokenPrice(100).
		SetPeriod(30).
		SetLevelLimit(50).
		SetUnitPrice(1.5).
		SetSlotMax(100).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.TemplateId() != 1000 {
		t.Errorf("model.TemplateId() = %d, want 1000", model.TemplateId())
	}
	if model.MesoPrice() != 500 {
		t.Errorf("model.MesoPrice() = %d, want 500", model.MesoPrice())
	}
	if model.DiscountRate() != 10 {
		t.Errorf("model.DiscountRate() = %d, want 10", model.DiscountRate())
	}
	if model.TokenTemplateId() != 2000 {
		t.Errorf("model.TokenTemplateId() = %d, want 2000", model.TokenTemplateId())
	}
	if model.TokenPrice() != 100 {
		t.Errorf("model.TokenPrice() = %d, want 100", model.TokenPrice())
	}
	if model.Period() != 30 {
		t.Errorf("model.Period() = %d, want 30", model.Period())
	}
	if model.LevelLimit() != 50 {
		t.Errorf("model.LevelLimit() = %d, want 50", model.LevelLimit())
	}
	if model.UnitPrice() != 1.5 {
		t.Errorf("model.UnitPrice() = %f, want 1.5", model.UnitPrice())
	}
	if model.SlotMax() != 100 {
		t.Errorf("model.SlotMax() = %d, want 100", model.SlotMax())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := commodities.NewModelBuilder().
		SetTemplateId(1000).
		SetMesoPrice(500).
		Build()

	if err != commodities.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestBuild_ZeroId(t *testing.T) {
	_, err := commodities.NewModelBuilder().
		SetId(uuid.Nil).
		SetTemplateId(1000).
		Build()

	if err != commodities.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()
	original, err := commodities.NewModelBuilder().
		SetId(id).
		SetTemplateId(1000).
		SetMesoPrice(500).
		SetDiscountRate(10).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := commodities.CloneModel(original).
		SetMesoPrice(600).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.MesoPrice() != 500 {
		t.Errorf("original.MesoPrice() = %d, want 500", original.MesoPrice())
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.Id() != id {
		t.Errorf("cloned.Id() = %v, want %v", cloned.Id(), id)
	}
	if cloned.TemplateId() != 1000 {
		t.Errorf("cloned.TemplateId() = %d, want 1000", cloned.TemplateId())
	}
	if cloned.MesoPrice() != 600 {
		t.Errorf("cloned.MesoPrice() = %d, want 600", cloned.MesoPrice())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	id := uuid.New()
	model := commodities.NewModelBuilder().
		SetId(id).
		SetTemplateId(1000).
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

	commodities.NewModelBuilder().
		SetTemplateId(1000).
		MustBuild() // Missing ID, should panic
}

func TestBuilderFluentChaining(t *testing.T) {
	id := uuid.New()
	model, err := commodities.NewModelBuilder().
		SetId(id).
		SetTemplateId(1000).
		SetMesoPrice(500).
		SetDiscountRate(10).
		SetTokenTemplateId(2000).
		SetTokenPrice(100).
		SetPeriod(30).
		SetLevelLimit(50).
		SetUnitPrice(1.5).
		SetSlotMax(100).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.SlotMax() != 100 {
		t.Errorf("model.SlotMax() = %d, want 100", model.SlotMax())
	}
}
