package asset_test

import (
	"atlas-inventory/asset"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewBuilder(t *testing.T) {
	id := uint32(123)
	compartmentId := uuid.New()
	templateId := uint32(456)
	referenceId := uint32(789)
	referenceType := asset.ReferenceTypeConsumable

	b := asset.NewBuilder[any](id, compartmentId, templateId, referenceId, referenceType)
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}

	m := b.Build()
	if m.Id() != id {
		t.Errorf("expected Id %d, got %d", id, m.Id())
	}
	if m.CompartmentId() != compartmentId {
		t.Errorf("expected CompartmentId %s, got %s", compartmentId, m.CompartmentId())
	}
	if m.TemplateId() != templateId {
		t.Errorf("expected TemplateId %d, got %d", templateId, m.TemplateId())
	}
	if m.ReferenceId() != referenceId {
		t.Errorf("expected ReferenceId %d, got %d", referenceId, m.ReferenceId())
	}
	if m.ReferenceType() != referenceType {
		t.Errorf("expected ReferenceType %s, got %s", referenceType, m.ReferenceType())
	}
	if m.Slot() != 0 {
		t.Errorf("expected default Slot 0, got %d", m.Slot())
	}
	if !m.Expiration().IsZero() {
		t.Errorf("expected zero Expiration, got %v", m.Expiration())
	}
}

func TestBuilderSetSlot(t *testing.T) {
	compartmentId := uuid.New()
	m := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeEquipable).
		SetSlot(5).
		Build()

	if m.Slot() != 5 {
		t.Errorf("expected Slot 5, got %d", m.Slot())
	}
}

func TestBuilderSetExpiration(t *testing.T) {
	compartmentId := uuid.New()
	expTime := time.Now().Add(24 * time.Hour)

	m := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeEquipable).
		SetExpiration(expTime).
		Build()

	if !m.Expiration().Equal(expTime) {
		t.Errorf("expected Expiration %v, got %v", expTime, m.Expiration())
	}
}

func TestBuilderSetReferenceData(t *testing.T) {
	compartmentId := uuid.New()
	refData := "test reference data"

	m := asset.NewBuilder[string](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable).
		SetReferenceData(refData).
		Build()

	if m.ReferenceData() != refData {
		t.Errorf("expected ReferenceData %s, got %s", refData, m.ReferenceData())
	}
}

func TestBuilderWithStructReferenceData(t *testing.T) {
	type TestRefData struct {
		Value int
		Name  string
	}

	compartmentId := uuid.New()
	refData := TestRefData{Value: 42, Name: "test"}

	m := asset.NewBuilder[TestRefData](1, compartmentId, 100, 200, asset.ReferenceTypeEtc).
		SetReferenceData(refData).
		Build()

	if m.ReferenceData().Value != 42 {
		t.Errorf("expected ReferenceData.Value 42, got %d", m.ReferenceData().Value)
	}
	if m.ReferenceData().Name != "test" {
		t.Errorf("expected ReferenceData.Name 'test', got %s", m.ReferenceData().Name)
	}
}

func TestClone(t *testing.T) {
	compartmentId := uuid.New()
	expTime := time.Now()

	original := asset.NewBuilder[string](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable).
		SetSlot(3).
		SetExpiration(expTime).
		SetReferenceData("original").
		Build()

	cloned := asset.Clone(original).Build()

	if cloned.Id() != original.Id() {
		t.Errorf("cloned Id %d != original Id %d", cloned.Id(), original.Id())
	}
	if cloned.CompartmentId() != original.CompartmentId() {
		t.Errorf("cloned CompartmentId %s != original CompartmentId %s", cloned.CompartmentId(), original.CompartmentId())
	}
	if cloned.TemplateId() != original.TemplateId() {
		t.Errorf("cloned TemplateId %d != original TemplateId %d", cloned.TemplateId(), original.TemplateId())
	}
	if cloned.ReferenceId() != original.ReferenceId() {
		t.Errorf("cloned ReferenceId %d != original ReferenceId %d", cloned.ReferenceId(), original.ReferenceId())
	}
	if cloned.ReferenceType() != original.ReferenceType() {
		t.Errorf("cloned ReferenceType %s != original ReferenceType %s", cloned.ReferenceType(), original.ReferenceType())
	}
	if cloned.Slot() != original.Slot() {
		t.Errorf("cloned Slot %d != original Slot %d", cloned.Slot(), original.Slot())
	}
	if !cloned.Expiration().Equal(original.Expiration()) {
		t.Errorf("cloned Expiration %v != original Expiration %v", cloned.Expiration(), original.Expiration())
	}
	if cloned.ReferenceData() != original.ReferenceData() {
		t.Errorf("cloned ReferenceData %s != original ReferenceData %s", cloned.ReferenceData(), original.ReferenceData())
	}
}

func TestCloneAndModify(t *testing.T) {
	compartmentId := uuid.New()
	original := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable).
		SetSlot(1).
		Build()

	modified := asset.Clone(original).
		SetSlot(5).
		Build()

	if original.Slot() != 1 {
		t.Errorf("original Slot changed: expected 1, got %d", original.Slot())
	}
	if modified.Slot() != 5 {
		t.Errorf("modified Slot incorrect: expected 5, got %d", modified.Slot())
	}
}

func TestFluentChaining(t *testing.T) {
	compartmentId := uuid.New()
	b := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeConsumable)

	// Verify each setter returns the builder for chaining
	result := b.SetSlot(1)
	if result != b {
		t.Error("SetSlot did not return the builder")
	}

	result = b.SetExpiration(time.Now())
	if result != b {
		t.Error("SetExpiration did not return the builder")
	}

	result = b.SetReferenceData(nil)
	if result != b {
		t.Error("SetReferenceData did not return the builder")
	}
}

func TestAllReferenceTypes(t *testing.T) {
	refTypes := []asset.ReferenceType{
		asset.ReferenceTypeEquipable,
		asset.ReferenceTypeCashEquipable,
		asset.ReferenceTypeConsumable,
		asset.ReferenceTypeSetup,
		asset.ReferenceTypeEtc,
		asset.ReferenceTypeCash,
		asset.ReferenceTypePet,
	}

	compartmentId := uuid.New()
	for _, refType := range refTypes {
		m := asset.NewBuilder[any](1, compartmentId, 100, 200, refType).Build()
		if m.ReferenceType() != refType {
			t.Errorf("expected ReferenceType %s, got %s", refType, m.ReferenceType())
		}
	}
}

func TestReferenceTypeChecks(t *testing.T) {
	compartmentId := uuid.New()

	tests := []struct {
		refType     asset.ReferenceType
		checkMethod string
		expected    bool
	}{
		{asset.ReferenceTypeEquipable, "IsEquipable", true},
		{asset.ReferenceTypeCashEquipable, "IsCashEquipable", true},
		{asset.ReferenceTypeConsumable, "IsConsumable", true},
		{asset.ReferenceTypeSetup, "IsSetup", true},
		{asset.ReferenceTypeEtc, "IsEtc", true},
		{asset.ReferenceTypeCash, "IsCash", true},
		{asset.ReferenceTypePet, "IsPet", true},
	}

	for _, tt := range tests {
		m := asset.NewBuilder[any](1, compartmentId, 100, 200, tt.refType).Build()

		var result bool
		switch tt.checkMethod {
		case "IsEquipable":
			result = m.IsEquipable()
		case "IsCashEquipable":
			result = m.IsCashEquipable()
		case "IsConsumable":
			result = m.IsConsumable()
		case "IsSetup":
			result = m.IsSetup()
		case "IsEtc":
			result = m.IsEtc()
		case "IsCash":
			result = m.IsCash()
		case "IsPet":
			result = m.IsPet()
		}

		if result != tt.expected {
			t.Errorf("%s() for %s: expected %v, got %v", tt.checkMethod, tt.refType, tt.expected, result)
		}
	}
}

func TestNegativeSlot(t *testing.T) {
	compartmentId := uuid.New()
	m := asset.NewBuilder[any](1, compartmentId, 100, 200, asset.ReferenceTypeEquipable).
		SetSlot(-5).
		Build()

	if m.Slot() != -5 {
		t.Errorf("expected negative Slot -5, got %d", m.Slot())
	}
}
