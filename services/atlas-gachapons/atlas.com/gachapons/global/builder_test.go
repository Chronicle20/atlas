package global_test

import (
	"atlas-gachapons/global"
	"atlas-gachapons/test"
	"testing"
)

func TestBuilderValidation(t *testing.T) {
	tenantId := test.TestTenantId

	t.Run("valid tier common", func(t *testing.T) {
		_, err := global.NewBuilder(tenantId, 0).
			SetItemId(1000).
			SetQuantity(1).
			SetTier("common").
			Build()
		if err != nil {
			t.Errorf("Expected no error for valid tier 'common', got: %v", err)
		}
	})

	t.Run("valid tier uncommon", func(t *testing.T) {
		_, err := global.NewBuilder(tenantId, 0).
			SetItemId(1000).
			SetQuantity(1).
			SetTier("uncommon").
			Build()
		if err != nil {
			t.Errorf("Expected no error for valid tier 'uncommon', got: %v", err)
		}
	})

	t.Run("valid tier rare", func(t *testing.T) {
		_, err := global.NewBuilder(tenantId, 0).
			SetItemId(1000).
			SetQuantity(1).
			SetTier("rare").
			Build()
		if err != nil {
			t.Errorf("Expected no error for valid tier 'rare', got: %v", err)
		}
	})

	t.Run("invalid tier", func(t *testing.T) {
		_, err := global.NewBuilder(tenantId, 0).
			SetItemId(1000).
			SetQuantity(1).
			SetTier("invalid").
			Build()
		if err == nil {
			t.Error("Expected error for invalid tier, got nil")
		}
	})

	t.Run("empty tier", func(t *testing.T) {
		_, err := global.NewBuilder(tenantId, 0).
			SetItemId(1000).
			SetQuantity(1).
			SetTier("").
			Build()
		if err == nil {
			t.Error("Expected error for empty tier, got nil")
		}
	})
}
