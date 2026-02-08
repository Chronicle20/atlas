package asset_test

import (
	"atlas-channel/asset"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := asset.NewModelBuilder(1, uuid.New(), 1000)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	compartmentId := uuid.New()
	expiration := time.Now().Add(24 * time.Hour)

	m, err := asset.NewModelBuilder(1, compartmentId, 1302000).
		SetSlot(5).
		SetExpiration(expiration).
		SetStrength(10).
		SetDexterity(20).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if m.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", m.Id())
	}
	if m.CompartmentId() != compartmentId {
		t.Errorf("model.CompartmentId() = %v, want %v", m.CompartmentId(), compartmentId)
	}
	if m.TemplateId() != 1302000 {
		t.Errorf("model.TemplateId() = %d, want 1302000", m.TemplateId())
	}
	if m.Slot() != 5 {
		t.Errorf("model.Slot() = %d, want 5", m.Slot())
	}
	if !m.IsEquipment() {
		t.Error("Expected IsEquipment() to return true for equip templateId")
	}
	if m.Strength() != 10 {
		t.Errorf("model.Strength() = %d, want 10", m.Strength())
	}
	if m.Dexterity() != 20 {
		t.Errorf("model.Dexterity() = %d, want 20", m.Dexterity())
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := asset.NewModelBuilder(0, uuid.New(), 1000).
		Build()

	if !errors.Is(err, asset.ErrInvalidId) {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	m := asset.NewModelBuilder(1, uuid.New(), 1000).MustBuild()

	if m.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", m.Id())
	}
}

func TestMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBuild() should have panicked on invalid input")
		}
	}()

	asset.NewModelBuilder(0, uuid.New(), 1000).MustBuild()
}

func TestCloneModel(t *testing.T) {
	original, _ := asset.NewModelBuilder(1, uuid.New(), 1302000).
		SetSlot(5).
		Build()

	cloned, err := asset.CloneModel(original).
		SetSlot(10).
		Build()

	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Slot() != 5 {
		t.Errorf("original.Slot() = %d, want 5", original.Slot())
	}

	// Cloned should have new slot
	if cloned.Slot() != 10 {
		t.Errorf("cloned.Slot() = %d, want 10", cloned.Slot())
	}
	// But preserve other fields
	if cloned.TemplateId() != 1302000 {
		t.Errorf("cloned.TemplateId() = %d, want 1302000", cloned.TemplateId())
	}
}

func TestBuilderWithDifferentInventoryTypes(t *testing.T) {
	tests := []struct {
		name       string
		templateId uint32
		checkFn    func(m asset.Model) bool
	}{
		{
			name:       "Consumable",
			templateId: 2000000,
			checkFn:    func(m asset.Model) bool { return m.IsConsumable() },
		},
		{
			name:       "Setup",
			templateId: 3000000,
			checkFn:    func(m asset.Model) bool { return m.IsSetup() },
		},
		{
			name:       "Etc",
			templateId: 4000000,
			checkFn:    func(m asset.Model) bool { return m.IsEtc() },
		},
		{
			name:       "Cash",
			templateId: 5000000,
			checkFn:    func(m asset.Model) bool { return m.IsCash() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := asset.NewModelBuilder(1, uuid.New(), tt.templateId).Build()
			if err != nil {
				t.Errorf("Build() with %s unexpected error: %v", tt.name, err)
			}
			if !tt.checkFn(m) {
				t.Errorf("Type check failed for %s", tt.name)
			}
		})
	}
}

func TestTypeChecks(t *testing.T) {
	equipable := asset.NewModelBuilder(1, uuid.New(), 1302000).MustBuild()
	if !equipable.IsEquipment() {
		t.Error("Expected IsEquipment() to return true for equip templateId")
	}

	consumable := asset.NewModelBuilder(2, uuid.New(), 2000000).MustBuild()
	if !consumable.IsConsumable() {
		t.Error("Expected IsConsumable() to return true for consumable templateId")
	}

	setup := asset.NewModelBuilder(3, uuid.New(), 3000000).MustBuild()
	if !setup.IsSetup() {
		t.Error("Expected IsSetup() to return true for setup templateId")
	}
}

// NewBuilder alias test for backward compatibility
func TestNewBuilderAlias(t *testing.T) {
	builder := asset.NewBuilder(uuid.New(), 1000)
	if builder == nil {
		t.Fatal("Expected NewBuilder alias to return initialized builder")
	}
}

// Clone alias test for backward compatibility
func TestCloneAlias(t *testing.T) {
	original := asset.NewModelBuilder(1, uuid.New(), 1302000).MustBuild()

	cloned := asset.Clone(original).SetSlot(10).MustBuild()

	if cloned.Slot() != 10 {
		t.Errorf("cloned.Slot() = %d, want 10", cloned.Slot())
	}
}

func TestPetModel(t *testing.T) {
	m := asset.NewModelBuilder(1, uuid.New(), 5000000).
		SetPetId(42).
		SetPetName("Fluffy").
		SetPetLevel(15).
		SetCloseness(100).
		SetFullness(80).
		SetPetSlot(0).
		SetCashId(99999).
		MustBuild()

	if !m.IsPet() {
		t.Error("Expected IsPet() to return true")
	}
	if m.PetId() != 42 {
		t.Errorf("PetId() = %d, want 42", m.PetId())
	}
	if m.PetName() != "Fluffy" {
		t.Errorf("PetName() = %s, want Fluffy", m.PetName())
	}
	if m.PetLevel() != 15 {
		t.Errorf("PetLevel() = %d, want 15", m.PetLevel())
	}
	if m.Closeness() != 100 {
		t.Errorf("Closeness() = %d, want 100", m.Closeness())
	}
	if m.Fullness() != 80 {
		t.Errorf("Fullness() = %d, want 80", m.Fullness())
	}
	if m.PetSlot() != 0 {
		t.Errorf("PetSlot() = %d, want 0", m.PetSlot())
	}
}

func TestQuantityBehavior(t *testing.T) {
	// Stackable items have explicit quantity
	consumable := asset.NewModelBuilder(1, uuid.New(), 2000000).
		SetQuantity(50).
		MustBuild()
	if consumable.Quantity() != 50 {
		t.Errorf("consumable.Quantity() = %d, want 50", consumable.Quantity())
	}
	if !consumable.HasQuantity() {
		t.Error("Expected HasQuantity() to return true for consumable")
	}

	// Equipment has implicit quantity of 1
	equip := asset.NewModelBuilder(2, uuid.New(), 1302000).MustBuild()
	if equip.Quantity() != 1 {
		t.Errorf("equip.Quantity() = %d, want 1", equip.Quantity())
	}
	if equip.HasQuantity() {
		t.Error("Expected HasQuantity() to return false for equipment")
	}
}
