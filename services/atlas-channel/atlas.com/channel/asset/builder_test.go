package asset_test

import (
	"atlas-channel/asset"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewModelBuilder(t *testing.T) {
	builder := asset.NewModelBuilder[asset.EquipableReferenceData](1, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable)
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	compartmentId := uuid.New()
	expiration := time.Now().Add(24 * time.Hour)
	refData := asset.EquipableReferenceData{}

	model, err := asset.NewModelBuilder[asset.EquipableReferenceData](1, compartmentId, 1000, 100, asset.ReferenceTypeEquipable).
		SetInventoryType(asset.InventoryTypeEquip).
		SetSlot(5).
		SetExpiration(expiration).
		SetReferenceData(refData).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.CompartmentId() != compartmentId {
		t.Errorf("model.CompartmentId() = %v, want %v", model.CompartmentId(), compartmentId)
	}
	if model.TemplateId() != 1000 {
		t.Errorf("model.TemplateId() = %d, want 1000", model.TemplateId())
	}
	if model.Slot() != 5 {
		t.Errorf("model.Slot() = %d, want 5", model.Slot())
	}
	if model.InventoryType() != asset.InventoryTypeEquip {
		t.Errorf("model.InventoryType() = %d, want %d", model.InventoryType(), asset.InventoryTypeEquip)
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := asset.NewModelBuilder[asset.EquipableReferenceData](0, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable).
		Build()

	if err != asset.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := asset.NewModelBuilder[asset.EquipableReferenceData](1, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable).MustBuild()

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

	asset.NewModelBuilder[asset.EquipableReferenceData](0, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable).MustBuild()
}

func TestCloneModel(t *testing.T) {
	original, _ := asset.NewModelBuilder[asset.EquipableReferenceData](1, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable).
		SetSlot(5).
		SetInventoryType(asset.InventoryTypeEquip).
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
	if cloned.TemplateId() != 1000 {
		t.Errorf("cloned.TemplateId() = %d, want 1000", cloned.TemplateId())
	}
}

func TestBuilderWithDifferentReferenceTypes(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ConsumableReferenceData",
			fn: func() error {
				_, err := asset.NewModelBuilder[asset.ConsumableReferenceData](1, uuid.New(), 2000, 200, asset.ReferenceTypeConsumable).
					SetInventoryType(asset.InventoryTypeUse).
					Build()
				return err
			},
		},
		{
			name: "SetupReferenceData",
			fn: func() error {
				_, err := asset.NewModelBuilder[asset.SetupReferenceData](1, uuid.New(), 3000, 300, asset.ReferenceTypeSetup).
					SetInventoryType(asset.InventoryTypeSetup).
					Build()
				return err
			},
		},
		{
			name: "EtcReferenceData",
			fn: func() error {
				_, err := asset.NewModelBuilder[asset.EtcReferenceData](1, uuid.New(), 4000, 400, asset.ReferenceTypeEtc).
					SetInventoryType(asset.InventoryTypeEtc).
					Build()
				return err
			},
		},
		{
			name: "CashReferenceData",
			fn: func() error {
				_, err := asset.NewModelBuilder[asset.CashReferenceData](1, uuid.New(), 5000, 500, asset.ReferenceTypeCash).
					SetInventoryType(asset.InventoryTypeCash).
					Build()
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("Build() with %s unexpected error: %v", tt.name, err)
			}
		})
	}
}

func TestReferenceTypeChecks(t *testing.T) {
	equipable := asset.NewModelBuilder[asset.EquipableReferenceData](1, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable).MustBuild()
	if !equipable.IsEquipable() {
		t.Error("Expected IsEquipable() to return true for equipable reference type")
	}

	consumable := asset.NewModelBuilder[asset.ConsumableReferenceData](2, uuid.New(), 2000, 200, asset.ReferenceTypeConsumable).MustBuild()
	if !consumable.IsConsumable() {
		t.Error("Expected IsConsumable() to return true for consumable reference type")
	}

	setup := asset.NewModelBuilder[asset.SetupReferenceData](3, uuid.New(), 3000, 300, asset.ReferenceTypeSetup).MustBuild()
	if !setup.IsSetup() {
		t.Error("Expected IsSetup() to return true for setup reference type")
	}
}

// NewBuilder alias test for backward compatibility
func TestNewBuilderAlias(t *testing.T) {
	builder := asset.NewBuilder[asset.EquipableReferenceData](1, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable)
	if builder == nil {
		t.Fatal("Expected NewBuilder alias to return initialized builder")
	}

	model := builder.MustBuild()
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
}

// Clone alias test for backward compatibility
func TestCloneAlias(t *testing.T) {
	original := asset.NewModelBuilder[asset.EquipableReferenceData](1, uuid.New(), 1000, 100, asset.ReferenceTypeEquipable).MustBuild()

	cloned := asset.Clone(original).SetSlot(10).MustBuild()

	if cloned.Slot() != 10 {
		t.Errorf("cloned.Slot() = %d, want 10", cloned.Slot())
	}
}
