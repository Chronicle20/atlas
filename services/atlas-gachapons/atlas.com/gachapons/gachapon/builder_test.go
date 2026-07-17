package gachapon_test

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/test"
	"testing"
)

func TestBuilderKind(t *testing.T) {
	tenantId := test.TestTenantId

	t.Run("defaults to gachapon when SetKind is never called", func(t *testing.T) {
		m, err := gachapon.NewBuilder(tenantId, "henesys").
			SetName("Henesys").
			SetNpcIds([]uint32{9100100}).
			SetCommonWeight(70).
			SetUncommonWeight(25).
			SetRareWeight(5).
			Build()
		if err != nil {
			t.Fatalf("Build() returned error: %v", err)
		}
		if m.Kind() != "gachapon" {
			t.Errorf("Expected default Kind() = %q, got %q", "gachapon", m.Kind())
		}
	})

	t.Run("SetKind overrides the default", func(t *testing.T) {
		m, err := gachapon.NewBuilder(tenantId, "pigmy-egg").
			SetName("Pigmy Egg").
			SetNpcIds([]uint32{9100100}).
			SetCommonWeight(70).
			SetUncommonWeight(25).
			SetRareWeight(5).
			SetKind("incubator").
			Build()
		if err != nil {
			t.Fatalf("Build() returned error: %v", err)
		}
		if m.Kind() != "incubator" {
			t.Errorf("Expected Kind() = %q, got %q", "incubator", m.Kind())
		}
	})
}
