package inventory_test

import (
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	"testing"

	inv "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := inventory.NewModelBuilder(100)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestNewBuilder_Alias(t *testing.T) {
	builder := inventory.NewBuilder(100)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	equipId := uuid.New()
	equip, _ := compartment.NewModelBuilder(equipId, 100, inv.TypeValueEquip, 24).Build()

	model, err := inventory.NewModelBuilder(100).
		SetEquipable(equip).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.CharacterId() != 100 {
		t.Errorf("model.CharacterId() = %d, want 100", model.CharacterId())
	}
	if model.Equipable().Id() != equipId {
		t.Errorf("model.Equipable().Id() = %v, want %v", model.Equipable().Id(), equipId)
	}
}

func TestBuild_MissingCharacterId(t *testing.T) {
	_, err := inventory.NewModelBuilder(0).Build()

	if err != inventory.ErrInvalidCharacterId {
		t.Errorf("Build() error = %v, want ErrInvalidCharacterId", err)
	}
}

func TestSetCompartment(t *testing.T) {
	compId := uuid.New()
	comp, _ := compartment.NewModelBuilder(compId, 100, inv.TypeValueUse, 24).Build()

	model, err := inventory.NewModelBuilder(100).
		SetCompartment(comp).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Consumable().Id() != compId {
		t.Errorf("model.Consumable().Id() = %v, want %v", model.Consumable().Id(), compId)
	}
}

func TestAllCompartments(t *testing.T) {
	equipId := uuid.New()
	useId := uuid.New()
	setupId := uuid.New()
	etcId := uuid.New()
	cashId := uuid.New()

	equip, _ := compartment.NewModelBuilder(equipId, 100, inv.TypeValueEquip, 24).Build()
	use, _ := compartment.NewModelBuilder(useId, 100, inv.TypeValueUse, 24).Build()
	setup, _ := compartment.NewModelBuilder(setupId, 100, inv.TypeValueSetup, 24).Build()
	etc, _ := compartment.NewModelBuilder(etcId, 100, inv.TypeValueETC, 24).Build()
	cash, _ := compartment.NewModelBuilder(cashId, 100, inv.TypeValueCash, 24).Build()

	model, err := inventory.NewModelBuilder(100).
		SetEquipable(equip).
		SetConsumable(use).
		SetSetup(setup).
		SetEtc(etc).
		SetCash(cash).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Equipable().Id() != equipId {
		t.Errorf("model.Equipable().Id() = %v, want %v", model.Equipable().Id(), equipId)
	}
	if model.Consumable().Id() != useId {
		t.Errorf("model.Consumable().Id() = %v, want %v", model.Consumable().Id(), useId)
	}
	if model.Setup().Id() != setupId {
		t.Errorf("model.Setup().Id() = %v, want %v", model.Setup().Id(), setupId)
	}
	if model.ETC().Id() != etcId {
		t.Errorf("model.ETC().Id() = %v, want %v", model.ETC().Id(), etcId)
	}
	if model.Cash().Id() != cashId {
		t.Errorf("model.Cash().Id() = %v, want %v", model.Cash().Id(), cashId)
	}
}

func TestCloneModel(t *testing.T) {
	equipId := uuid.New()
	equip, _ := compartment.NewModelBuilder(equipId, 100, inv.TypeValueEquip, 24).Build()

	original, err := inventory.NewModelBuilder(100).
		SetEquipable(equip).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	useId := uuid.New()
	use, _ := compartment.NewModelBuilder(useId, 100, inv.TypeValueUse, 24).Build()

	cloned, err := inventory.CloneModel(original).
		SetConsumable(use).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Cloned should have both compartments
	if cloned.Equipable().Id() != equipId {
		t.Errorf("cloned.Equipable().Id() = %v, want %v", cloned.Equipable().Id(), equipId)
	}
	if cloned.Consumable().Id() != useId {
		t.Errorf("cloned.Consumable().Id() = %v, want %v", cloned.Consumable().Id(), useId)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := inventory.NewModelBuilder(100).MustBuild()

	if model.CharacterId() != 100 {
		t.Errorf("model.CharacterId() = %d, want 100", model.CharacterId())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	inventory.NewModelBuilder(0).MustBuild() // Zero character ID, should panic
}

func TestBuilderSupplier(t *testing.T) {
	supplier := inventory.BuilderSupplier(100)
	builder, err := supplier()
	if err != nil {
		t.Fatalf("BuilderSupplier() unexpected error: %v", err)
	}
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestFoldCompartment(t *testing.T) {
	compId := uuid.New()
	comp, _ := compartment.NewModelBuilder(compId, 100, inv.TypeValueEquip, 24).Build()

	builder := inventory.NewBuilder(100)
	builder, err := inventory.FoldCompartment(builder, comp)
	if err != nil {
		t.Fatalf("FoldCompartment() unexpected error: %v", err)
	}

	model, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Equipable().Id() != compId {
		t.Errorf("model.Equipable().Id() = %v, want %v", model.Equipable().Id(), compId)
	}
}
