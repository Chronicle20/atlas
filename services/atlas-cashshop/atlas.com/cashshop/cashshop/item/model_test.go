package item

import (
	"testing"
	"time"
)

func TestBuilderCreatesModel(t *testing.T) {
	id := uint32(1)
	cashId := int64(1234567890)
	templateId := uint32(5000001)
	quantity := uint32(10)
	flag := uint16(1)
	purchasedBy := uint32(12345)
	expiration := time.Now().Add(30 * 24 * time.Hour)

	m := NewBuilder().
		SetId(id).
		SetCashId(cashId).
		SetTemplateId(templateId).
		SetQuantity(quantity).
		SetFlag(flag).
		SetPurchasedBy(purchasedBy).
		SetExpiration(expiration).
		Build()

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
	if !m.Expiration().Equal(expiration) {
		t.Errorf("Expiration mismatch: expected %v, got %v", expiration, m.Expiration())
	}
}

func TestBuilderFluentInterface(t *testing.T) {
	// Verify fluent interface returns the same builder
	b := NewBuilder()

	if b.SetId(1) != b {
		t.Error("SetId should return the same builder")
	}
	if b.SetCashId(123) != b {
		t.Error("SetCashId should return the same builder")
	}
	if b.SetTemplateId(5000) != b {
		t.Error("SetTemplateId should return the same builder")
	}
	if b.SetQuantity(1) != b {
		t.Error("SetQuantity should return the same builder")
	}
	if b.SetFlag(0) != b {
		t.Error("SetFlag should return the same builder")
	}
	if b.SetPurchasedBy(12345) != b {
		t.Error("SetPurchasedBy should return the same builder")
	}
	if b.SetExpiration(time.Now()) != b {
		t.Error("SetExpiration should return the same builder")
	}
}

func TestBuilderDefaultValues(t *testing.T) {
	// Build with no values set
	m := NewBuilder().Build()

	if m.Id() != 0 {
		t.Errorf("Default Id should be 0, got %d", m.Id())
	}
	if m.CashId() != 0 {
		t.Errorf("Default CashId should be 0, got %d", m.CashId())
	}
	if m.TemplateId() != 0 {
		t.Errorf("Default TemplateId should be 0, got %d", m.TemplateId())
	}
	if m.Quantity() != 0 {
		t.Errorf("Default Quantity should be 0, got %d", m.Quantity())
	}
	if m.Flag() != 0 {
		t.Errorf("Default Flag should be 0, got %d", m.Flag())
	}
	if m.PurchasedBy() != 0 {
		t.Errorf("Default PurchasedBy should be 0, got %d", m.PurchasedBy())
	}
}

func TestModelImmutability(t *testing.T) {
	// Create a model
	m := NewBuilder().
		SetId(1).
		SetCashId(123).
		SetTemplateId(5000).
		SetQuantity(10).
		Build()

	// Accessor methods should return consistent values
	id1 := m.Id()
	id2 := m.Id()
	if id1 != id2 {
		t.Error("Model accessors should return consistent values")
	}

	cashId1 := m.CashId()
	cashId2 := m.CashId()
	if cashId1 != cashId2 {
		t.Error("Model accessors should return consistent values")
	}
}

func TestMultipleBuildsFromSameBuilder(t *testing.T) {
	b := NewBuilder().
		SetId(1).
		SetTemplateId(5000)

	m1 := b.Build()
	m2 := b.Build()

	// Both builds should produce equivalent models
	if m1.Id() != m2.Id() {
		t.Error("Multiple builds from same builder should produce equivalent models")
	}
	if m1.TemplateId() != m2.TemplateId() {
		t.Error("Multiple builds from same builder should produce equivalent models")
	}
}
