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

// TestKarmaUsedLeavesSpikesAlone is the FR-4.5 guard for atlas-cashshop: 0x02
// is FlagSpikes on an EQUIP, and both EquipableReferenceData and
// CashEquipableReferenceData are equip-class by type with the karma bit
// hardcoded to FlagKarmaEquip (never resolved through KarmaFlagFor) — which
// makes them the path MOST exposed to a wrong-bit write, not exempt from the
// guard. A karma mark on an unspiked equip must not render spikes, and
// clearing karma on a spiked equip must not silently clear FlagSpikes.
func TestKarmaUsedLeavesSpikesAlone(t *testing.T) {
	t.Run("EquipableReferenceData", func(t *testing.T) {
		plain := NewEquipableReferenceDataBuilder().SetKarmaUsed(true).Build()
		if plain.HasSpikes() {
			t.Fatal("HasSpikes() = true after karma-marking an unspiked equip; the wrong bit was written")
		}

		spiked := NewEquipableReferenceDataBuilder().SetSpikes(true).SetKarmaUsed(true).Build()
		if !spiked.HasSpikes() {
			t.Fatal("HasSpikes() = false after karma-marking a spiked equip")
		}
		if !spiked.IsKarmaUsed() {
			t.Fatal("IsKarmaUsed() = false on a spiked equip")
		}

		cleared := NewEquipableReferenceDataBuilder().Clone(spiked).SetKarmaUsed(false).Build()
		if !cleared.HasSpikes() {
			t.Fatal("clearing karma on a spiked equip cleared FlagSpikes")
		}
	})

	t.Run("CashEquipableReferenceData", func(t *testing.T) {
		plain := NewCashEquipableReferenceDataBuilder().SetKarmaUsed(true).Build()
		if plain.HasSpikes() {
			t.Fatal("HasSpikes() = true after karma-marking an unspiked equip; the wrong bit was written")
		}

		spiked := NewCashEquipableReferenceDataBuilder().SetSpikes(true).SetKarmaUsed(true).Build()
		if !spiked.HasSpikes() {
			t.Fatal("HasSpikes() = false after karma-marking a spiked equip")
		}
		if !spiked.IsKarmaUsed() {
			t.Fatal("IsKarmaUsed() = false on a spiked equip")
		}

		cleared := NewCashEquipableReferenceDataBuilder().Clone(spiked).SetKarmaUsed(false).Build()
		if !cleared.HasSpikes() {
			t.Fatal("clearing karma on a spiked equip cleared FlagSpikes")
		}
	})
}
