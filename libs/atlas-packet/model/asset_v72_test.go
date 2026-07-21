package model

import (
	"testing"
	"time"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestEquipableLegacyTrailerTiers pins the three legacy equip-trailer tiers,
// all IDA-verified from GW_ItemSlotEquip::RawDecode:
//
//   - v48 (@0x49c332) / v61 (@0x4b4e7d): base+slots+level+15 stats+owner+flag,
//     then a single 8-byte buffer (non-cash). NO levelType/level/exp trailer.
//   - v72 (@0x4d0172): adds levelType(1)+level(1)+exp(4) before that buffer and a
//     second 8-byte buffer + int(4) after it → +18 bytes vs v61.
//   - v79 (@0x4d7ee8) / v83 (@0x4e3c3d): adds hammersApplied(4) → +4 bytes vs v72.
//
// atlas historically wrote the v79 layout for every version, over-writing each
// equipped item on v48/v61/v72 and disconnecting the client on channel entry.
func TestEquipableLegacyTrailerTiers(t *testing.T) {
	a := NewAsset(false, -11, 1302000, time.Time{}).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001) // hammersApplied = 3
	if !a.IsEquipment() || a.IsCashEquipment() {
		t.Fatalf("fixture must be a non-cash equip (isEquip=%v isCash=%v)", a.IsEquipment(), a.IsCashEquipment())
	}

	enc := func(major uint16) []byte {
		return pt.Encode(t, pt.CreateContext("GMS", major, 1), a.Encode, nil)
	}
	v48, v61, v72, v79 := len(enc(48)), len(enc(61)), len(enc(72)), len(enc(79))

	if v48 != v61 {
		t.Errorf("non-cash equip: v48 len %d != v61 len %d (both are the pre-72 legacy layout)", v48, v61)
	}
	if v72 != v61+18 {
		t.Errorf("non-cash equip: v72 len %d, v61 len %d; want v72 == v61 + 18 (levelType+level+exp+2nd buf+int)", v72, v61)
	}
	if v79 != v72+4 {
		t.Errorf("non-cash equip: v79 len %d, v72 len %d; want v79 == v72 + 4 (hammersApplied)", v79, v72)
	}
}

// TestCashEquipableLegacyTrailerTiers mirrors the above for cash equips. A cash
// item skips the non-cash 8-byte buffer, so v48/v61 read NOTHING after the flag
// short; v72 adds 6 filler + Int64 + int32 (=18); v79 widens the filler to 10 (+4).
func TestCashEquipableLegacyTrailerTiers(t *testing.T) {
	a := NewAsset(false, -11, 1302000, time.Time{}).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0).
		SetCashId(1)
	if !a.IsCashEquipment() {
		t.Fatalf("fixture must be a cash equip (isCashEquip=%v)", a.IsCashEquipment())
	}

	enc := func(major uint16) []byte {
		return pt.Encode(t, pt.CreateContext("GMS", major, 1), a.Encode, nil)
	}
	v48, v61, v72, v79 := len(enc(48)), len(enc(61)), len(enc(72)), len(enc(79))

	if v48 != v61 {
		t.Errorf("cash equip: v48 len %d != v61 len %d (both read nothing after flag)", v48, v61)
	}
	if v72 != v61+18 {
		t.Errorf("cash equip: v72 len %d, v61 len %d; want v72 == v61 + 18 (6 filler + Int64 + int32)", v72, v61)
	}
	if v79 != v72+4 {
		t.Errorf("cash equip: v79 len %d, v72 len %d; want v79 == v72 + 4 (10 vs 6 filler)", v79, v72)
	}
}
