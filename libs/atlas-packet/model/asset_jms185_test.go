package model

import (
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=model/Asset version=jms_v185 ida=0x50feb9

// TestAssetJMSEquipableLength pins the JMS non-cash equip block at 98 bytes
// (2-byte slot prefix + 96-byte item body), matching the captured
// SetField(0x007b) block size from bug-jms185-naked-avatar-red-equips.md
// (root cause 2 evidence). Field order is aligned against
// GW_ItemSlotEquip::RawDecode @0x50feb9 (JMS 185) / @0x4f8360 (GMS v95).
func TestAssetJMSEquipableLength(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("JMS", 185, 1)
	a := NewAsset(false, 1, 1002357, time.Time{}).
		SetEquipmentStats(4, 4, 4, 4, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)

	got := a.Encode(l, ctx)(nil)
	const wantLen = 98
	if len(got) != wantLen {
		t.Fatalf("jms non-cash equip length = %d, want %d", len(got), wantLen)
	}
}

// TestAssetJMSEquipableDurabilityMinusOne asserts the Decode4 immediately
// after nEXP on the JMS equip arm is nDurability, sent as -1 ("no
// durability") — matching what the GMS v84+ arm already does at the same
// position. Atlas previously wrote hammersApplied (always 0 for these items)
// here, which the client reads as "broken" (red overlay, unworn on the
// avatar). See root cause 2 in bug-jms185-naked-avatar-red-equips.md.
func TestAssetJMSEquipableDurabilityMinusOne(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("JMS", 185, 1)
	a := NewAsset(false, 1, 1002357, time.Time{}).
		SetEquipmentStats(4, 4, 4, 4, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 99, 0) // hammersApplied=99: must NOT reach the wire on JMS

	got := a.Encode(l, ctx)(nil)
	// Offset of nDurability within the 98-byte block: slot(2) + type(1) +
	// templateId(4) + isCash(1) + expire(8) + slots(1) + level(1) +
	// JMS-extra(1) + stats(30) + owner(2) + flag(2) + levelType(1) +
	// level(1) + experience(4) = 59.
	const durabilityOffset = 59
	if len(got) < durabilityOffset+4 {
		t.Fatalf("encoded body too short: %d bytes", len(got))
	}
	durability := int32(got[durabilityOffset]) | int32(got[durabilityOffset+1])<<8 |
		int32(got[durabilityOffset+2])<<16 | int32(got[durabilityOffset+3])<<24
	if durability != -1 {
		t.Errorf("nDurability at offset %d = %d, want -1", durabilityOffset, durability)
	}
}

// TestAssetJMSCashEquipableLength pins the JMS cash equip block at 98 bytes,
// including the 15-byte JMS trailer (Decode1 + Decode2×5 + Decode4) that
// GW_ItemSlotEquip::RawDecode @0x50feb9 reads unconditionally for cash and
// non-cash equips alike — only the first trailing DecodeBuffer(8) is gated on
// liCashItemSN == 0. Without it, any JMS character holding a cash equip
// desyncs CharacterData by 15 bytes (latent defect 1,
// bug-jms185-naked-avatar-red-equips.md).
func TestAssetJMSCashEquipableLength(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("JMS", 185, 1)
	a := NewAsset(false, 1, 1002357, time.Time{}).
		SetCashId(555).
		SetEquipmentStats(4, 4, 4, 4, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)

	got := a.Encode(l, ctx)(nil)
	const wantLen = 98
	if len(got) != wantLen {
		t.Fatalf("jms cash equip length = %d, want %d", len(got), wantLen)
	}
}

// TestAssetJMSEquipRoundTrip round-trips both the non-cash and cash JMS equip
// arms through Decode, confirming the added JMS-only bytes are consumed
// (rather than merely written) and the following stream position stays
// aligned.
func TestAssetJMSEquipRoundTrip(t *testing.T) {
	ctx := test.CreateContext("JMS", 185, 1)

	t.Run("non-cash", func(t *testing.T) {
		// zeroPosition=true: Asset.Decode does not itself read the leading
		// equip slot prefix (that is done by the caller, e.g.
		// decodeEquipmentSection in character/data.go); omitting it here
		// keeps this a pure item-body round-trip.
		in := NewAsset(true, 1, 1002357, time.Time{}).
			SetEquipmentStats(4, 4, 4, 4, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0).
			SetEquipmentMeta(7, 0, 0, 0, 0, 0)
		test.RoundTrip(t, ctx, in.Encode, (&Asset{}).Decode, nil)
	})

	t.Run("cash", func(t *testing.T) {
		in := NewAsset(true, 1, 1002357, time.Time{}).
			SetCashId(555).
			SetEquipmentStats(4, 4, 4, 4, 50, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0).
			SetEquipmentMeta(7, 0, 0, 0, 0, 0)
		test.RoundTrip(t, ctx, in.Encode, (&Asset{}).Decode, nil)
	})
}
