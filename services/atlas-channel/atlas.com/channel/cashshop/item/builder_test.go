package item_test

import (
	"atlas-channel/cashshop/item"
	"testing"
	"time"
)

func TestNewModelBuilder(t *testing.T) {
	builder := item.NewModelBuilder()
	if builder == nil {
		t.Fatal("Expected builder to be initialized")
	}
}

func TestBuild_AllFieldsSet(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)
	model, err := item.NewModelBuilder().
		SetId(1).
		SetCashId(12345).
		SetTemplateId(5000000).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(100).
		SetExpiration(expiration).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.Id() != 1 {
		t.Errorf("model.Id() = %d, want 1", model.Id())
	}
	if model.CashId() != 12345 {
		t.Errorf("model.CashId() = %d, want 12345", model.CashId())
	}
	if model.TemplateId() != 5000000 {
		t.Errorf("model.TemplateId() = %d, want 5000000", model.TemplateId())
	}
	if model.Quantity() != 1 {
		t.Errorf("model.Quantity() = %d, want 1", model.Quantity())
	}
	if model.Flag() != 0 {
		t.Errorf("model.Flag() = %d, want 0", model.Flag())
	}
	if model.PurchasedBy() != 100 {
		t.Errorf("model.PurchasedBy() = %d, want 100", model.PurchasedBy())
	}
	if !model.Expiration().Equal(expiration) {
		t.Errorf("model.Expiration() = %v, want %v", model.Expiration(), expiration)
	}
}

func TestBuild_MissingId(t *testing.T) {
	_, err := item.NewModelBuilder().
		SetCashId(12345).
		SetTemplateId(5000000).
		Build()

	if err != item.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestBuild_ZeroId(t *testing.T) {
	_, err := item.NewModelBuilder().
		SetId(0).
		SetTemplateId(5000000).
		Build()

	if err != item.ErrInvalidId {
		t.Errorf("Build() error = %v, want ErrInvalidId", err)
	}
}

func TestCloneModel(t *testing.T) {
	original, err := item.NewModelBuilder().
		SetId(1).
		SetCashId(12345).
		SetTemplateId(5000000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	cloned, err := item.CloneModel(original).
		SetQuantity(5).
		Build()
	if err != nil {
		t.Fatalf("CloneModel().Build() unexpected error: %v", err)
	}

	// Original should be unchanged
	if original.Quantity() != 1 {
		t.Errorf("original.Quantity() = %d, want 1", original.Quantity())
	}

	// Cloned should have new values but preserve unchanged fields
	if cloned.Id() != 1 {
		t.Errorf("cloned.Id() = %d, want 1", cloned.Id())
	}
	if cloned.CashId() != 12345 {
		t.Errorf("cloned.CashId() = %d, want 12345", cloned.CashId())
	}
	if cloned.TemplateId() != 5000000 {
		t.Errorf("cloned.TemplateId() = %d, want 5000000", cloned.TemplateId())
	}
	if cloned.Quantity() != 5 {
		t.Errorf("cloned.Quantity() = %d, want 5", cloned.Quantity())
	}
}

func TestMustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustBuild() panicked unexpectedly: %v", r)
		}
	}()

	model := item.NewModelBuilder().
		SetId(1).
		SetTemplateId(5000000).
		MustBuild()

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

	item.NewModelBuilder().
		SetTemplateId(5000000).
		MustBuild() // Missing ID, should panic
}

func TestBuilderFluentChaining(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)
	model, err := item.NewModelBuilder().
		SetId(1).
		SetCashId(12345).
		SetTemplateId(5000000).
		SetQuantity(1).
		SetFlag(0).
		SetPurchasedBy(100).
		SetExpiration(expiration).
		Build()

	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if model.TemplateId() != 5000000 {
		t.Errorf("model.TemplateId() = %d, want 5000000", model.TemplateId())
	}
}
