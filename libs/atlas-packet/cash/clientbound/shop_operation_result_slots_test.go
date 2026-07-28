package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v61 ida=0x462bcb
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v72 ida=0x4727ba
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v79 ida=0x473a7f
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v83 ida=0x47ab2e
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v84 ida=0x47dccc
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v87 ida=0x4862ee
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=gms_v95 ida=0x494ed0
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountSuccess version=jms_v185 ida=0x48d682
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v48 ida=0x454f4f
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v61 ida=0x462ca2
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v72 ida=0x472896
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v79 ida=0x473b5b
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v83 ida=0x47ac0a
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v84 ida=0x47dda8
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v87 ida=0x4863d3
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=gms_v95 ida=0x494f70
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountSuccess version=jms_v185 ida=0x48d75e
// packet-audit:verify packet=cash/clientbound/CashIncBuyCharacterCountSuccess version=gms_v95 ida=0x495000
// packet-audit:verify packet=cash/clientbound/CashIncBuyCharacterCountSuccess version=jms_v185 ida=0x48d82f
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtSuccess version=gms_v79 ida=0x473c2c
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtSuccess version=gms_v83 ida=0x47acdb
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtSuccess version=gms_v84 ida=0x47de79
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtSuccess version=gms_v87 ida=0x4864ad
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtSuccess version=gms_v95 ida=0x497490
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtSuccess version=jms_v185 ida=0x48d8b1

// Per-version dispatcher mode bytes for the counter-arm family (task-183 Wave
// 1.2/3), taken from docs/tasks/task-183-cashshop-result-family/arm-catalog.md.
// Legacy keys (GMS/v48/v61/v72/v79) added in Wave 3 batch MISC-L.

// incTrunkCountSuccessModes: n-a in GMS/v48 (arm-catalog.md — feature does not
// exist yet in v48; no case in the switch, func_query confirms 0 hits).
var incTrunkCountSuccessModes = map[string]byte{
	"GMS/v61": 0x46, "GMS/v72": 0x4E, "GMS/v79": 0x5A,
	"GMS/v83": 0x62, "GMS/v84": 0x65, "GMS/v87": 0x67, "GMS/v95": 0x6F, "JMS/v185": 0x63,
}

var incCharacterSlotCountSuccessModes = map[string]byte{
	"GMS/v48": 0x3F, "GMS/v61": 0x48, "GMS/v72": 0x50, "GMS/v79": 0x5C,
	"GMS/v83": 0x64, "GMS/v84": 0x67, "GMS/v87": 0x69, "GMS/v95": 0x71, "JMS/v185": 0x65,
}

// incBuyCharacterCountSuccessModes: n-a in GMS/v48/v61/v79/v83/v84/v87 (only
// present starting v95, per catalog, among MODERN + the versions verified
// here). GMS/v72 IS present (mode 0x52) but is DELIBERATELY OMITTED from this
// map — task-183 Wave 3 §3 confirms v72's arm is a MATERIALLY DIFFERENT wire
// shape (slotIndex:Decode2 + GW_ItemSlotBase::Decode, a locker-item-consuming
// operation) than this struct's bare mode+uint16 counter. See
// .superpowers/sdd/task-3.4-legacy-misc-report.md for the decompiled v72 read
// order — reported, not verified; the controller will gate the codec (a
// separate v72-shaped type) before this cell can be verified.
var incBuyCharacterCountSuccessModes = map[string]byte{
	"GMS/v95": 0x73, "JMS/v185": 0x67,
}

// enableEquipSlotExtSuccessModes: n-a in GMS/v48/v61/v72 (feature does not
// exist in those builds — arm-catalog.md).
var enableEquipSlotExtSuccessModes = map[string]byte{
	"GMS/v79": 0x5E,
	"GMS/v83": 0x66, "GMS/v84": 0x69, "GMS/v87": 0x6B, "GMS/v95": 0x75, "JMS/v185": 0x69,
}

func TestIncTrunkCountSuccessByteFixture(t *testing.T) {
	const trunkCount = uint16(0x0010)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := incTrunkCountSuccessModes[variantKey(v)]
			if !ok {
				t.Skipf("no INC_TRUNK_COUNT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewIncTrunkCountSuccess(mode, trunkCount)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(trunkCount), byte(trunkCount >> 8)}
			if !bytesEqual(got, want) {
				t.Errorf("INC_TRUNK_COUNT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := IncTrunkCountSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.TrunkCount() != trunkCount {
				t.Errorf("round-trip mismatch: mode %v trunkCount %v", output.Mode(), output.TrunkCount())
			}
		})
	}
}

func TestIncCharacterSlotCountSuccessByteFixture(t *testing.T) {
	const slotCount = uint16(0x0004)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := incCharacterSlotCountSuccessModes[variantKey(v)]
			if !ok {
				t.Skipf("no INC_CHARACTER_SLOT_COUNT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewIncCharacterSlotCountSuccess(mode, slotCount)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(slotCount), byte(slotCount >> 8)}
			if !bytesEqual(got, want) {
				t.Errorf("INC_CHARACTER_SLOT_COUNT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := IncCharacterSlotCountSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.SlotCount() != slotCount {
				t.Errorf("round-trip mismatch: mode %v slotCount %v", output.Mode(), output.SlotCount())
			}
		})
	}
}

func TestIncBuyCharacterCountSuccessByteFixture(t *testing.T) {
	const buyCharacterCount = uint16(0x0002)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := incBuyCharacterCountSuccessModes[variantKey(v)]
			if !ok {
				t.Skipf("no INC_BUY_CHARACTER_COUNT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewIncBuyCharacterCountSuccess(mode, buyCharacterCount)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(buyCharacterCount), byte(buyCharacterCount >> 8)}
			if !bytesEqual(got, want) {
				t.Errorf("INC_BUY_CHARACTER_COUNT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := IncBuyCharacterCountSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.BuyCharacterCount() != buyCharacterCount {
				t.Errorf("round-trip mismatch: mode %v buyCharacterCount %v", output.Mode(), output.BuyCharacterCount())
			}
		})
	}
}

func TestEnableEquipSlotExtSuccessByteFixture(t *testing.T) {
	const slotIndex, days = uint16(0x0003), uint16(0x001E)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := enableEquipSlotExtSuccessModes[variantKey(v)]
			if !ok {
				t.Skipf("no ENABLE_EQUIP_SLOT_EXT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEnableEquipSlotExtSuccess(mode, slotIndex, days)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(slotIndex), byte(slotIndex >> 8), byte(days), byte(days >> 8)}
			if !bytesEqual(got, want) {
				t.Errorf("ENABLE_EQUIP_SLOT_EXT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := EnableEquipSlotExtSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.SlotIndex() != slotIndex || output.Days() != days {
				t.Errorf("round-trip mismatch: mode %v slotIndex %v days %v", output.Mode(), output.SlotIndex(), output.Days())
			}
		})
	}
}
