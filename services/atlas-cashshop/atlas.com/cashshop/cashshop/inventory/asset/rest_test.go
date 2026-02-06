package asset

import (
	"atlas-cashshop/cashshop/item"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransform(t *testing.T) {
	// Create test data
	assetId := uuid.New()
	compartmentId := uuid.New()

	// Create an item
	testItem := item.NewBuilder().
		SetId(1).
		SetCashId(1001).
		SetTemplateId(5000).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(12345).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Create asset model
	m := NewBuilder(assetId, compartmentId, testItem).Build()

	// Transform to REST model
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify fields
	if rm.Id != assetId {
		t.Errorf("Id mismatch: expected %v, got %v", assetId, rm.Id)
	}
	if rm.CompartmentId != compartmentId {
		t.Errorf("CompartmentId mismatch: expected %v, got %v", compartmentId, rm.CompartmentId)
	}
	if rm.Item.Id != testItem.Id() {
		t.Errorf("Item.Id mismatch: expected %d, got %d", testItem.Id(), rm.Item.Id)
	}
	if rm.Item.TemplateId != testItem.TemplateId() {
		t.Errorf("Item.TemplateId mismatch: expected %d, got %d", testItem.TemplateId(), rm.Item.TemplateId)
	}
}

func TestExtract(t *testing.T) {
	// Create test data
	assetId := uuid.New()
	compartmentId := uuid.New()

	// Create REST item model
	itemRm := item.RestModel{
		Id:          1,
		CashId:      1001,
		TemplateId:  5000,
		Quantity:    1,
		Flag:        0,
		PurchasedBy: 12345,
	}

	// Create REST model
	rm := RestModel{
		Id:            assetId,
		CompartmentId: compartmentId,
		Item:          itemRm,
	}

	// Extract to domain model
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify fields
	if m.Id() != assetId {
		t.Errorf("Id mismatch: expected %v, got %v", assetId, m.Id())
	}
	if m.CompartmentId() != compartmentId {
		t.Errorf("CompartmentId mismatch: expected %v, got %v", compartmentId, m.CompartmentId())
	}
	if m.Item().Id() != itemRm.Id {
		t.Errorf("Item.Id mismatch: expected %d, got %d", itemRm.Id, m.Item().Id())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	// Create test data
	assetId := uuid.New()
	compartmentId := uuid.New()

	// Create an item
	testItem := item.NewBuilder().
		SetId(1).
		SetCashId(1001).
		SetTemplateId(5000).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(12345).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Create asset model
	original := NewBuilder(assetId, compartmentId, testItem).Build()

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
	if original.CompartmentId() != result.CompartmentId() {
		t.Errorf("CompartmentId mismatch after round-trip: expected %v, got %v", original.CompartmentId(), result.CompartmentId())
	}
	if original.Item().Id() != result.Item().Id() {
		t.Errorf("Item.Id mismatch after round-trip: expected %d, got %d", original.Item().Id(), result.Item().Id())
	}
	if original.Item().TemplateId() != result.Item().TemplateId() {
		t.Errorf("Item.TemplateId mismatch after round-trip: expected %d, got %d", original.Item().TemplateId(), result.Item().TemplateId())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	expected := "assets"
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

func TestAssetDelegatesMethods(t *testing.T) {
	// Create an item
	testItem := item.NewBuilder().
		SetId(1).
		SetCashId(1001).
		SetTemplateId(5000).
		SetQuantity(10).
		SetFlag(0).
		SetPurchasedBy(12345).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Create asset model
	assetId := uuid.New()
	compartmentId := uuid.New()
	m := NewBuilder(assetId, compartmentId, testItem).Build()

	// Verify delegate methods
	if m.TemplateId() != testItem.TemplateId() {
		t.Errorf("TemplateId mismatch: expected %d, got %d", testItem.TemplateId(), m.TemplateId())
	}
	if m.Quantity() != testItem.Quantity() {
		t.Errorf("Quantity mismatch: expected %d, got %d", testItem.Quantity(), m.Quantity())
	}
}
