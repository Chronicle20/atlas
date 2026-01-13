package compartment_test

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"testing"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
)

func TestNewBuilder(t *testing.T) {
	id := uuid.New()
	characterId := uint32(123)
	invType := inventory.TypeValueUse
	capacity := uint32(24)

	b := compartment.NewBuilder(id, characterId, invType, capacity)
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}

	m := b.Build()
	if m.Id() != id {
		t.Errorf("expected Id %s, got %s", id, m.Id())
	}
	if m.CharacterId() != characterId {
		t.Errorf("expected CharacterId %d, got %d", characterId, m.CharacterId())
	}
	if m.Type() != invType {
		t.Errorf("expected Type %d, got %d", invType, m.Type())
	}
	if m.Capacity() != capacity {
		t.Errorf("expected Capacity %d, got %d", capacity, m.Capacity())
	}
	if len(m.Assets()) != 0 {
		t.Errorf("expected empty assets, got %d", len(m.Assets()))
	}
}

func TestBuilderSetCapacity(t *testing.T) {
	id := uuid.New()
	b := compartment.NewBuilder(id, 1, inventory.TypeValueEquip, 24)

	m := b.SetCapacity(48).Build()

	if m.Capacity() != 48 {
		t.Errorf("expected Capacity 48, got %d", m.Capacity())
	}
}

func TestBuilderAddAsset(t *testing.T) {
	compartmentId := uuid.New()
	b := compartment.NewBuilder(compartmentId, 1, inventory.TypeValueUse, 24)

	// Create a test asset
	testAsset := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable).
		SetSlot(1).
		Build()

	m := b.AddAsset(testAsset).Build()

	if len(m.Assets()) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(m.Assets()))
	}
	if m.Assets()[0].Id() != 1 {
		t.Errorf("expected asset Id 1, got %d", m.Assets()[0].Id())
	}
}

func TestBuilderAddMultipleAssets(t *testing.T) {
	compartmentId := uuid.New()
	b := compartment.NewBuilder(compartmentId, 1, inventory.TypeValueUse, 24)

	asset1 := asset.NewBuilder[any](1, compartmentId, 100, 201, asset.ReferenceTypeConsumable).SetSlot(1).Build()
	asset2 := asset.NewBuilder[any](2, compartmentId, 100, 202, asset.ReferenceTypeConsumable).SetSlot(2).Build()

	m := b.AddAsset(asset1).AddAsset(asset2).Build()

	if len(m.Assets()) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(m.Assets()))
	}
}

func TestBuilderSetAssets(t *testing.T) {
	compartmentId := uuid.New()
	b := compartment.NewBuilder(compartmentId, 1, inventory.TypeValueUse, 24)

	// Add initial asset
	initialAsset := asset.NewBuilder[any](1, compartmentId, 100, 201, asset.ReferenceTypeConsumable).SetSlot(1).Build()
	b.AddAsset(initialAsset)

	// Replace with new assets
	newAsset1 := asset.NewBuilder[any](10, compartmentId, 100, 210, asset.ReferenceTypeConsumable).SetSlot(1).Build()
	newAsset2 := asset.NewBuilder[any](11, compartmentId, 100, 211, asset.ReferenceTypeConsumable).SetSlot(2).Build()

	m := b.SetAssets([]asset.Model[any]{newAsset1, newAsset2}).Build()

	if len(m.Assets()) != 2 {
		t.Fatalf("expected 2 assets after SetAssets, got %d", len(m.Assets()))
	}
	if m.Assets()[0].Id() != 10 {
		t.Errorf("expected first asset Id 10, got %d", m.Assets()[0].Id())
	}
}

func TestClone(t *testing.T) {
	id := uuid.New()
	characterId := uint32(123)
	invType := inventory.TypeValueETC
	capacity := uint32(32)

	original := compartment.NewBuilder(id, characterId, invType, capacity).Build()
	cloned := compartment.Clone(original).Build()

	if cloned.Id() != original.Id() {
		t.Errorf("cloned Id %s != original Id %s", cloned.Id(), original.Id())
	}
	if cloned.CharacterId() != original.CharacterId() {
		t.Errorf("cloned CharacterId %d != original CharacterId %d", cloned.CharacterId(), original.CharacterId())
	}
	if cloned.Type() != original.Type() {
		t.Errorf("cloned Type %d != original Type %d", cloned.Type(), original.Type())
	}
	if cloned.Capacity() != original.Capacity() {
		t.Errorf("cloned Capacity %d != original Capacity %d", cloned.Capacity(), original.Capacity())
	}
}

func TestCloneWithAssets(t *testing.T) {
	compartmentId := uuid.New()
	testAsset := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable).SetSlot(1).Build()

	original := compartment.NewBuilder(compartmentId, 1, inventory.TypeValueUse, 24).
		AddAsset(testAsset).
		Build()

	cloned := compartment.Clone(original).Build()

	if len(cloned.Assets()) != len(original.Assets()) {
		t.Errorf("cloned assets count %d != original assets count %d", len(cloned.Assets()), len(original.Assets()))
	}
}

func TestCloneAndModify(t *testing.T) {
	id := uuid.New()
	original := compartment.NewBuilder(id, 1, inventory.TypeValueEquip, 24).Build()

	modified := compartment.Clone(original).SetCapacity(48).Build()

	if original.Capacity() != 24 {
		t.Errorf("original Capacity changed: expected 24, got %d", original.Capacity())
	}
	if modified.Capacity() != 48 {
		t.Errorf("modified Capacity incorrect: expected 48, got %d", modified.Capacity())
	}
}

func TestFluentChaining(t *testing.T) {
	id := uuid.New()
	compartmentId := uuid.New()
	b := compartment.NewBuilder(id, 1, inventory.TypeValueUse, 24)

	// Verify each setter returns the builder for chaining
	result := b.SetCapacity(32)
	if result != b {
		t.Error("SetCapacity did not return the builder")
	}

	testAsset := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable).Build()
	result = b.AddAsset(testAsset)
	if result != b {
		t.Error("AddAsset did not return the builder")
	}

	result = b.SetAssets([]asset.Model[any]{})
	if result != b {
		t.Error("SetAssets did not return the builder")
	}
}

func TestAllInventoryTypes(t *testing.T) {
	types := []inventory.Type{
		inventory.TypeValueEquip,
		inventory.TypeValueUse,
		inventory.TypeValueSetup,
		inventory.TypeValueETC,
		inventory.TypeValueCash,
	}

	for _, invType := range types {
		id := uuid.New()
		m := compartment.NewBuilder(id, 1, invType, 24).Build()
		if m.Type() != invType {
			t.Errorf("expected Type %d, got %d", invType, m.Type())
		}
	}
}
