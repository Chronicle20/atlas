package item_test

import (
	"atlas-gachapons/item"
	"atlas-gachapons/test"
	"testing"
)

func TestBuilderValidation(t *testing.T) {
	tenantId := test.TestTenantId

	t.Run("valid tier common", func(t *testing.T) {
		_, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("common").
			Build()
		if err != nil {
			t.Errorf("Expected no error for valid tier 'common', got: %v", err)
		}
	})

	t.Run("valid tier uncommon", func(t *testing.T) {
		_, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("uncommon").
			Build()
		if err != nil {
			t.Errorf("Expected no error for valid tier 'uncommon', got: %v", err)
		}
	})

	t.Run("valid tier rare", func(t *testing.T) {
		_, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("rare").
			Build()
		if err != nil {
			t.Errorf("Expected no error for valid tier 'rare', got: %v", err)
		}
	})

	t.Run("invalid tier", func(t *testing.T) {
		_, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("invalid").
			Build()
		if err == nil {
			t.Error("Expected error for invalid tier, got nil")
		}
	})

	t.Run("empty tier", func(t *testing.T) {
		_, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("").
			Build()
		if err == nil {
			t.Error("Expected error for empty tier, got nil")
		}
	})
}

func TestBuilderWeight(t *testing.T) {
	tenantId := test.TestTenantId

	t.Run("defaults to 0 when SetWeight is never called", func(t *testing.T) {
		m, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("common").
			Build()
		if err != nil {
			t.Fatalf("Build() returned error: %v", err)
		}
		if m.Weight() != 0 {
			t.Errorf("Expected default Weight() = 0, got %d", m.Weight())
		}
	})

	t.Run("SetWeight overrides the default", func(t *testing.T) {
		m, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("gachapon-1").
			SetItemId(1000).
			SetQuantity(1).
			SetTier("common").
			SetWeight(50).
			Build()
		if err != nil {
			t.Fatalf("Build() returned error: %v", err)
		}
		if m.Weight() != 50 {
			t.Errorf("Expected Weight() = 50, got %d", m.Weight())
		}
	})
}
