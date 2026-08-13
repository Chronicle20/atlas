package asset

import "testing"

// TestKarmaUsedRoundTrip is the FR-4.4 regression guard for atlas-cashshop.
// EquipableReferenceData and CashEquipableReferenceData carry no template id
// (unlike the six asset models), but are equip-class BY TYPE, so IsKarmaUsed
// unconditionally reads FlagKarmaEquip (0x10) — the same bit SetKarmaUsed has
// always written. Before task-223 the getter read FlagKarmaUse (0x02) instead,
// so a set never read back. Equip-only: cashshop has no bundle-shaped
// reference type.
func TestKarmaUsedRoundTrip(t *testing.T) {
	t.Run("EquipableReferenceData", func(t *testing.T) {
		m := NewEquipableReferenceDataBuilder().SetKarmaUsed(true).Build()
		if !m.IsKarmaUsed() {
			t.Fatal("IsKarmaUsed() = false after SetKarmaUsed(true)")
		}
		cleared := NewEquipableReferenceDataBuilder().Clone(m).SetKarmaUsed(false).Build()
		if cleared.IsKarmaUsed() {
			t.Fatal("IsKarmaUsed() = true after SetKarmaUsed(false)")
		}
	})

	t.Run("CashEquipableReferenceData", func(t *testing.T) {
		m := NewCashEquipableReferenceDataBuilder().SetKarmaUsed(true).Build()
		if !m.IsKarmaUsed() {
			t.Fatal("IsKarmaUsed() = false after SetKarmaUsed(true)")
		}
		cleared := NewCashEquipableReferenceDataBuilder().Clone(m).SetKarmaUsed(false).Build()
		if cleared.IsKarmaUsed() {
			t.Fatal("IsKarmaUsed() = true after SetKarmaUsed(false)")
		}
	})
}
