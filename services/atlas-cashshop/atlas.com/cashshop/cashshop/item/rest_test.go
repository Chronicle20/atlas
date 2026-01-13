package item

import (
	"testing"
	"time"
)

func TestTransform(t *testing.T) {
	// Create an item model using direct struct creation (within package)
	id := uint32(1)
	cashId := int64(1234567890)
	templateId := uint32(5000001)
	quantity := uint32(10)
	flag := uint16(0)
	purchasedBy := uint32(12345)
	expiration := time.Now().Add(30 * 24 * time.Hour)

	m := Model{
		id:          id,
		cashId:      cashId,
		templateId:  templateId,
		quantity:    quantity,
		flag:        flag,
		purchasedBy: purchasedBy,
		expiration:  expiration,
	}

	// Transform to REST model
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify all fields are correctly transformed
	if rm.Id != id {
		t.Errorf("Id mismatch: expected %d, got %d", id, rm.Id)
	}
	if rm.CashId != cashId {
		t.Errorf("CashId mismatch: expected %d, got %d", cashId, rm.CashId)
	}
	if rm.TemplateId != templateId {
		t.Errorf("TemplateId mismatch: expected %d, got %d", templateId, rm.TemplateId)
	}
	if rm.Quantity != quantity {
		t.Errorf("Quantity mismatch: expected %d, got %d", quantity, rm.Quantity)
	}
	if rm.Flag != flag {
		t.Errorf("Flag mismatch: expected %d, got %d", flag, rm.Flag)
	}
	if rm.PurchasedBy != purchasedBy {
		t.Errorf("PurchasedBy mismatch: expected %d, got %d", purchasedBy, rm.PurchasedBy)
	}
}

func TestExtract(t *testing.T) {
	// Create a REST model
	id := uint32(1)
	cashId := int64(1234567890)
	templateId := uint32(5000001)
	quantity := uint32(10)
	flag := uint16(0)
	purchasedBy := uint32(12345)

	rm := RestModel{
		Id:          id,
		CashId:      cashId,
		TemplateId:  templateId,
		Quantity:    quantity,
		Flag:        flag,
		PurchasedBy: purchasedBy,
	}

	// Extract to domain model
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify all fields are correctly extracted
	if m.Id() != id {
		t.Errorf("Id mismatch: expected %d, got %d", id, m.Id())
	}
	if m.CashId() != cashId {
		t.Errorf("CashId mismatch: expected %d, got %d", cashId, m.CashId())
	}
	if m.TemplateId() != templateId {
		t.Errorf("TemplateId mismatch: expected %d, got %d", templateId, m.TemplateId())
	}
	if m.Quantity() != quantity {
		t.Errorf("Quantity mismatch: expected %d, got %d", quantity, m.Quantity())
	}
	if m.Flag() != flag {
		t.Errorf("Flag mismatch: expected %d, got %d", flag, m.Flag())
	}
	if m.PurchasedBy() != purchasedBy {
		t.Errorf("PurchasedBy mismatch: expected %d, got %d", purchasedBy, m.PurchasedBy())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	// Create an item model
	id := uint32(1)
	cashId := int64(1234567890)
	templateId := uint32(5000001)
	quantity := uint32(10)
	flag := uint16(0)
	purchasedBy := uint32(12345)

	original := Model{
		id:          id,
		cashId:      cashId,
		templateId:  templateId,
		quantity:    quantity,
		flag:        flag,
		purchasedBy: purchasedBy,
	}

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

	// Verify round-trip preserves all values
	if original.Id() != result.Id() {
		t.Errorf("Id mismatch after round-trip: expected %d, got %d", original.Id(), result.Id())
	}
	if original.CashId() != result.CashId() {
		t.Errorf("CashId mismatch after round-trip: expected %d, got %d", original.CashId(), result.CashId())
	}
	if original.TemplateId() != result.TemplateId() {
		t.Errorf("TemplateId mismatch after round-trip: expected %d, got %d", original.TemplateId(), result.TemplateId())
	}
	if original.Quantity() != result.Quantity() {
		t.Errorf("Quantity mismatch after round-trip: expected %d, got %d", original.Quantity(), result.Quantity())
	}
	if original.Flag() != result.Flag() {
		t.Errorf("Flag mismatch after round-trip: expected %d, got %d", original.Flag(), result.Flag())
	}
	if original.PurchasedBy() != result.PurchasedBy() {
		t.Errorf("PurchasedBy mismatch after round-trip: expected %d, got %d", original.PurchasedBy(), result.PurchasedBy())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	expected := "items"
	if rm.GetName() != expected {
		t.Errorf("GetName mismatch: expected %s, got %s", expected, rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	rm := RestModel{Id: 12345}
	expected := "12345"
	if rm.GetID() != expected {
		t.Errorf("GetID mismatch: expected %s, got %s", expected, rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("12345")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if rm.Id != 12345 {
		t.Errorf("SetID mismatch: expected 12345, got %d", rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("not-a-number")
	if err == nil {
		t.Error("SetID should fail for non-numeric input")
	}
}
