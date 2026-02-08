package asset

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransform(t *testing.T) {
	// Create test data
	compartmentId := uuid.New()

	// Create asset model directly (flattened)
	m := NewBuilder(compartmentId, 5000).
		SetId(1).
		SetCashId(1001).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(12345).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Transform to REST model
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify fields
	if rm.Id != m.Id() {
		t.Errorf("Id mismatch: expected %d, got %d", m.Id(), rm.Id)
	}
	if rm.CompartmentId != compartmentId.String() {
		t.Errorf("CompartmentId mismatch: expected %v, got %v", compartmentId.String(), rm.CompartmentId)
	}
	if rm.TemplateId != 5000 {
		t.Errorf("TemplateId mismatch: expected %d, got %d", 5000, rm.TemplateId)
	}
	if rm.CashId != 1001 {
		t.Errorf("CashId mismatch: expected %d, got %d", 1001, rm.CashId)
	}
}

func TestExtract(t *testing.T) {
	// Create REST model
	rm := RestModel{
		Id:            1,
		CompartmentId: uuid.New().String(),
		CashId:        1001,
		TemplateId:    5000,
		CommodityId:   0,
		Quantity:      1,
		Flag:          0,
		PurchasedBy:   12345,
	}

	// Extract to domain model
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify fields
	if m.Id() != rm.Id {
		t.Errorf("Id mismatch: expected %d, got %d", rm.Id, m.Id())
	}
	if m.TemplateId() != rm.TemplateId {
		t.Errorf("TemplateId mismatch: expected %d, got %d", rm.TemplateId, m.TemplateId())
	}
	if m.CashId() != rm.CashId {
		t.Errorf("CashId mismatch: expected %d, got %d", rm.CashId, m.CashId())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	// Create asset model
	compartmentId := uuid.New()
	original := NewBuilder(compartmentId, 5000).
		SetId(1).
		SetCashId(1001).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(12345).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

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
		t.Errorf("Id mismatch after round-trip: expected %d, got %d", original.Id(), result.Id())
	}
	if original.TemplateId() != result.TemplateId() {
		t.Errorf("TemplateId mismatch after round-trip: expected %d, got %d", original.TemplateId(), result.TemplateId())
	}
	if original.CashId() != result.CashId() {
		t.Errorf("CashId mismatch after round-trip: expected %d, got %d", original.CashId(), result.CashId())
	}
	if original.Quantity() != result.Quantity() {
		t.Errorf("Quantity mismatch after round-trip: expected %d, got %d", original.Quantity(), result.Quantity())
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
	rm := RestModel{Id: 42}
	expected := "42"
	if rm.GetID() != expected {
		t.Errorf("GetID mismatch: expected %s, got %s", expected, rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("42")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if rm.Id != 42 {
		t.Errorf("SetID mismatch: expected %d, got %d", 42, rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("not-a-number")
	if err == nil {
		t.Error("SetID should fail for invalid number")
	}
}

func TestAssetFlatFields(t *testing.T) {
	// Create asset model with all fields
	compartmentId := uuid.New()
	m := NewBuilder(compartmentId, 5000).
		SetId(1).
		SetCashId(1001).
		SetQuantity(10).
		SetFlag(0).
		SetPurchasedBy(12345).
		SetExpiration(time.Now().Add(30 * 24 * time.Hour)).
		Build()

	// Verify all field accessors
	if m.TemplateId() != 5000 {
		t.Errorf("TemplateId mismatch: expected %d, got %d", 5000, m.TemplateId())
	}
	if m.Quantity() != 10 {
		t.Errorf("Quantity mismatch: expected %d, got %d", 10, m.Quantity())
	}
	if m.CashId() != 1001 {
		t.Errorf("CashId mismatch: expected %d, got %d", 1001, m.CashId())
	}
	if m.CompartmentId() != compartmentId {
		t.Errorf("CompartmentId mismatch: expected %v, got %v", compartmentId, m.CompartmentId())
	}
}
