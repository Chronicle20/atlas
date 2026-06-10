package model

import (
	"testing"
	"time"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestEquipableV84ExtraInt pins the v84+ equip field. v84's
// GW_ItemSlotEquip::RawDecode reads one extra int (offset +224) after
// hammersApplied that v83's older inline equip decode does not. atlas wrote the
// v83 layout for every version, so a v84 client under-ran each equipped item by
// 4 bytes (×4 starting equips on a fresh character) → ZException → silent
// disconnect entering the channel. The encoded equip must therefore be exactly
// 4 bytes longer for GMS v84+ than for v83.
func TestEquipableV84ExtraInt(t *testing.T) {
	a := NewAsset(false, -11, 1302000, time.Time{}).
		SetEquipmentStats(10, 10, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	if !a.IsEquipment() || a.IsCashEquipment() {
		t.Fatalf("fixture must be a non-cash equip (isEquip=%v isCash=%v)", a.IsEquipment(), a.IsCashEquipment())
	}

	enc := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, a.Encode, nil)
	}
	v83 := enc(83)
	for _, major := range []uint16{84, 85, 86, 87} {
		if got := enc(major); len(got) != len(v83)+4 {
			t.Errorf("equip GMS v%d encoded len %d; want v83 len %d + 4 (RawDecode +224)", major, len(got), len(v83))
		}
	}
}
