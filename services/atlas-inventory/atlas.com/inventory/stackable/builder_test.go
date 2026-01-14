package stackable_test

import (
	"atlas-inventory/stackable"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	b := stackable.NewBuilder()
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}
	m := b.Build()
	if m.Id() != 0 {
		t.Errorf("expected Id 0, got %d", m.Id())
	}
	if m.Quantity() != 0 {
		t.Errorf("expected Quantity 0, got %d", m.Quantity())
	}
	if m.OwnerId() != 0 {
		t.Errorf("expected OwnerId 0, got %d", m.OwnerId())
	}
	if m.Flag() != 0 {
		t.Errorf("expected Flag 0, got %d", m.Flag())
	}
	if m.Rechargeable() != 0 {
		t.Errorf("expected Rechargeable 0, got %d", m.Rechargeable())
	}
}

func TestBuilderSetters(t *testing.T) {
	m := stackable.NewBuilder().
		SetID(123).
		SetQuantity(50).
		SetOwnerId(456).
		SetFlag(789).
		SetRechargeable(100).
		Build()

	if m.Id() != 123 {
		t.Errorf("expected Id 123, got %d", m.Id())
	}
	if m.Quantity() != 50 {
		t.Errorf("expected Quantity 50, got %d", m.Quantity())
	}
	if m.OwnerId() != 456 {
		t.Errorf("expected OwnerId 456, got %d", m.OwnerId())
	}
	if m.Flag() != 789 {
		t.Errorf("expected Flag 789, got %d", m.Flag())
	}
	if m.Rechargeable() != 100 {
		t.Errorf("expected Rechargeable 100, got %d", m.Rechargeable())
	}
}

func TestClone(t *testing.T) {
	original := stackable.NewBuilder().
		SetID(1).
		SetQuantity(10).
		SetOwnerId(20).
		SetFlag(30).
		SetRechargeable(40).
		Build()

	cloned := stackable.Clone(original).Build()

	if cloned.Id() != original.Id() {
		t.Errorf("cloned Id %d != original Id %d", cloned.Id(), original.Id())
	}
	if cloned.Quantity() != original.Quantity() {
		t.Errorf("cloned Quantity %d != original Quantity %d", cloned.Quantity(), original.Quantity())
	}
	if cloned.OwnerId() != original.OwnerId() {
		t.Errorf("cloned OwnerId %d != original OwnerId %d", cloned.OwnerId(), original.OwnerId())
	}
	if cloned.Flag() != original.Flag() {
		t.Errorf("cloned Flag %d != original Flag %d", cloned.Flag(), original.Flag())
	}
	if cloned.Rechargeable() != original.Rechargeable() {
		t.Errorf("cloned Rechargeable %d != original Rechargeable %d", cloned.Rechargeable(), original.Rechargeable())
	}
}

func TestCloneAndModify(t *testing.T) {
	original := stackable.NewBuilder().
		SetID(1).
		SetQuantity(10).
		Build()

	modified := stackable.Clone(original).
		SetQuantity(20).
		Build()

	if original.Quantity() != 10 {
		t.Errorf("original Quantity changed: expected 10, got %d", original.Quantity())
	}
	if modified.Quantity() != 20 {
		t.Errorf("modified Quantity incorrect: expected 20, got %d", modified.Quantity())
	}
	if modified.Id() != original.Id() {
		t.Errorf("modified Id changed: expected %d, got %d", original.Id(), modified.Id())
	}
}

func TestFluentChaining(t *testing.T) {
	b := stackable.NewBuilder()

	// Verify each setter returns the builder for chaining
	result := b.SetID(1)
	if result != b {
		t.Error("SetID did not return the builder")
	}

	result = b.SetQuantity(1)
	if result != b {
		t.Error("SetQuantity did not return the builder")
	}

	result = b.SetOwnerId(1)
	if result != b {
		t.Error("SetOwnerId did not return the builder")
	}

	result = b.SetFlag(1)
	if result != b {
		t.Error("SetFlag did not return the builder")
	}

	result = b.SetRechargeable(1)
	if result != b {
		t.Error("SetRechargeable did not return the builder")
	}
}
