package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/item"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestTransform(t *testing.T) {
	// Create test data
	compartmentId := uuid.New()
	accountId := uint32(12345)
	capacity := uint32(100)

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
	testAsset := asset.NewBuilder(uuid.New(), compartmentId, testItem).Build()

	// Create compartment model
	m := NewBuilder(compartmentId, accountId, TypeExplorer, capacity).
		AddAsset(testAsset).
		Build()

	// Transform to REST model
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify fields
	if rm.Id != compartmentId {
		t.Errorf("Id mismatch: expected %v, got %v", compartmentId, rm.Id)
	}
	if rm.AccountId != accountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", accountId, rm.AccountId)
	}
	if rm.Type != TypeExplorer {
		t.Errorf("Type mismatch: expected %d, got %d", TypeExplorer, rm.Type)
	}
	if rm.Capacity != capacity {
		t.Errorf("Capacity mismatch: expected %d, got %d", capacity, rm.Capacity)
	}
	if len(rm.Assets) != 1 {
		t.Errorf("Assets count mismatch: expected 1, got %d", len(rm.Assets))
	}
}

func TestExtract(t *testing.T) {
	// Create test data
	compartmentId := uuid.New()
	accountId := uint32(12345)
	capacity := uint32(100)

	// Create REST model
	rm := RestModel{
		Id:        compartmentId,
		AccountId: accountId,
		Type:      TypeCygnus,
		Capacity:  capacity,
		Assets:    []asset.RestModel{},
	}

	// Extract to domain model
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify fields
	if m.Id() != compartmentId {
		t.Errorf("Id mismatch: expected %v, got %v", compartmentId, m.Id())
	}
	if m.AccountId() != accountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", accountId, m.AccountId())
	}
	if m.Type() != TypeCygnus {
		t.Errorf("Type mismatch: expected %d, got %d", TypeCygnus, m.Type())
	}
	if m.Capacity() != capacity {
		t.Errorf("Capacity mismatch: expected %d, got %d", capacity, m.Capacity())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	// Create test data
	compartmentId := uuid.New()
	accountId := uint32(12345)
	capacity := uint32(100)

	// Create compartment model
	original := NewBuilder(compartmentId, accountId, TypeLegend, capacity).Build()

	// Transform to REST model
	rm, err := Transform(original)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Extract back to domain model
	result, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify round-trip preserves values
	if original.Id() != result.Id() {
		t.Errorf("Id mismatch after round-trip: expected %v, got %v", original.Id(), result.Id())
	}
	if original.AccountId() != result.AccountId() {
		t.Errorf("AccountId mismatch after round-trip: expected %d, got %d", original.AccountId(), result.AccountId())
	}
	if original.Type() != result.Type() {
		t.Errorf("Type mismatch after round-trip: expected %d, got %d", original.Type(), result.Type())
	}
	if original.Capacity() != result.Capacity() {
		t.Errorf("Capacity mismatch after round-trip: expected %d, got %d", original.Capacity(), result.Capacity())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	expected := "compartments"
	if rm.GetName() != expected {
		t.Errorf("GetName mismatch: expected %s, got %s", expected, rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	id := uuid.New()
	rm := RestModel{Id: id}
	expected := id.String()
	if rm.GetID() != expected {
		t.Errorf("GetID mismatch: expected %s, got %s", expected, rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	id := uuid.New()
	err := rm.SetID(id.String())
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if rm.Id != id {
		t.Errorf("SetID mismatch: expected %v, got %v", id, rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("not-a-valid-uuid")
	if err == nil {
		t.Error("SetID should fail for invalid UUID")
	}
}

func TestCompartmentTypes(t *testing.T) {
	if TypeExplorer != CompartmentType(1) {
		t.Errorf("TypeExplorer should be 1, got %d", TypeExplorer)
	}
	if TypeCygnus != CompartmentType(2) {
		t.Errorf("TypeCygnus should be 2, got %d", TypeCygnus)
	}
	if TypeLegend != CompartmentType(3) {
		t.Errorf("TypeLegend should be 3, got %d", TypeLegend)
	}
}
