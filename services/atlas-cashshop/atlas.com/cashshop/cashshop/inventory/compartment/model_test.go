package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/item"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestBuilderCreatesModel(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)
	compartmentType := TypeExplorer
	capacity := uint32(100)

	m := NewBuilder(id, accountId, compartmentType, capacity).Build()

	if m.Id() != id {
		t.Errorf("Id mismatch: expected %v, got %v", id, m.Id())
	}
	if m.AccountId() != accountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", accountId, m.AccountId())
	}
	if m.Type() != compartmentType {
		t.Errorf("Type mismatch: expected %d, got %d", compartmentType, m.Type())
	}
	if m.Capacity() != capacity {
		t.Errorf("Capacity mismatch: expected %d, got %d", capacity, m.Capacity())
	}
	if len(m.Assets()) != 0 {
		t.Errorf("Assets should be empty initially, got %d", len(m.Assets()))
	}
}

func TestBuilderWithAssets(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)

	// Create an item
	testItem := item.NewBuilder().
		SetId(1).
		SetCashId(1001).
		SetTemplateId(5000).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(accountId).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Create an asset
	testAsset := asset.NewBuilder(uuid.New(), id, testItem).Build()

	m := NewBuilder(id, accountId, TypeCygnus, 100).
		AddAsset(testAsset).
		Build()

	if len(m.Assets()) != 1 {
		t.Errorf("Assets count mismatch: expected 1, got %d", len(m.Assets()))
	}
}

func TestBuilderSetAssets(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)

	// Create items and assets
	item1 := item.NewBuilder().
		SetId(1).
		SetTemplateId(5000).
		Build()

	item2 := item.NewBuilder().
		SetId(2).
		SetTemplateId(5001).
		Build()

	asset1 := asset.NewBuilder(uuid.New(), id, item1).Build()
	asset2 := asset.NewBuilder(uuid.New(), id, item2).Build()

	assets := []asset.Model{asset1, asset2}

	m := NewBuilder(id, accountId, TypeLegend, 100).
		SetAssets(assets).
		Build()

	if len(m.Assets()) != 2 {
		t.Errorf("Assets count mismatch: expected 2, got %d", len(m.Assets()))
	}
}

func TestBuilderSetCapacity(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)

	m := NewBuilder(id, accountId, TypeExplorer, 50).
		SetCapacity(100).
		Build()

	if m.Capacity() != 100 {
		t.Errorf("Capacity mismatch: expected 100, got %d", m.Capacity())
	}
}

func TestBuilderFluentInterface(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)
	b := NewBuilder(id, accountId, TypeExplorer, 50)

	if b.SetCapacity(100) != b {
		t.Error("SetCapacity should return the same builder")
	}

	testItem := item.NewBuilder().SetId(1).Build()
	testAsset := asset.NewBuilder(uuid.New(), id, testItem).Build()

	if b.AddAsset(testAsset) != b {
		t.Error("AddAsset should return the same builder")
	}
}

func TestCloneModel(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)

	original := NewBuilder(id, accountId, TypeCygnus, 100).Build()

	cloned := Clone(original)
	modified := cloned.SetCapacity(200).Build()

	// Original should be unchanged
	if original.Capacity() != 100 {
		t.Errorf("Original capacity should be unchanged, got %d", original.Capacity())
	}

	// Modified should have new capacity
	if modified.Capacity() != 200 {
		t.Errorf("Modified capacity should be 200, got %d", modified.Capacity())
	}
}

func TestFindById(t *testing.T) {
	compartmentId := uuid.New()
	accountId := uint32(12345)

	// Create items and assets
	assetId := uuid.New()
	testItem := item.NewBuilder().
		SetId(1).
		SetTemplateId(5000).
		Build()
	testAsset := asset.NewBuilder(assetId, compartmentId, testItem).Build()

	m := NewBuilder(compartmentId, accountId, TypeExplorer, 100).
		AddAsset(testAsset).
		Build()

	// Find existing asset
	found, ok := m.FindById(assetId)
	if !ok {
		t.Error("Should find existing asset by ID")
	}
	if found.Id() != assetId {
		t.Errorf("Found asset ID mismatch: expected %v, got %v", assetId, found.Id())
	}

	// Find non-existing asset
	_, ok = m.FindById(uuid.New())
	if ok {
		t.Error("Should not find non-existing asset")
	}
}

func TestFindByTemplateId(t *testing.T) {
	compartmentId := uuid.New()
	accountId := uint32(12345)

	// Create items and assets
	templateId := uint32(5000)
	testItem := item.NewBuilder().
		SetId(1).
		SetTemplateId(templateId).
		Build()
	testAsset := asset.NewBuilder(uuid.New(), compartmentId, testItem).Build()

	m := NewBuilder(compartmentId, accountId, TypeExplorer, 100).
		AddAsset(testAsset).
		Build()

	// Find existing asset by template ID
	found, ok := m.FindByTemplateId(templateId)
	if !ok {
		t.Error("Should find existing asset by template ID")
	}
	if found.TemplateId() != templateId {
		t.Errorf("Found asset template ID mismatch: expected %d, got %d", templateId, found.TemplateId())
	}

	// Find non-existing asset
	_, ok = m.FindByTemplateId(99999)
	if ok {
		t.Error("Should not find non-existing asset by template ID")
	}
}
