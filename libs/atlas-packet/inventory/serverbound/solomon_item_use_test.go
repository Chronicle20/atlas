package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v72 ida=0x90cb20
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v79 ida=0x95dee8
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v83 ida=0xa12685
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v84 ida=0xa5cac2
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v87 ida=0xaa80ac
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v92 ida=0x9b0ab0
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=gms_v95 ida=0x9db1c0
// packet-audit:verify packet=inventory/serverbound/InventorySolomonItemUse version=jms_v185 ida=0xaf883d
func TestSolomonItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SolomonItemUse{ItemUse: ItemUse{operation: CharacterItemUseSolomonHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 2370000}}
			output := NewSolomonItemUse()
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Operation() != CharacterItemUseSolomonHandle {
				t.Errorf("operation: got %v, want %v", output.Operation(), CharacterItemUseSolomonHandle)
			}
		})
	}
}

// TestSolomonItemUseByteFixtureV95 pins the USE_SOLOMON_ITEM wire against
// CWvsContext::SendExpUpItemUseRequest (v95 @0x9db1c0): the client emits
// Encode4(get_update_time()) + Encode2(nPOS) + Encode4(nItemID). The body is
// invariant across all eight in-scope columns (gms_v72..gms_v95, jms_v185).
func TestSolomonItemUseByteFixtureV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	in := SolomonItemUse{ItemUse: ItemUse{operation: CharacterItemUseSolomonHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 2370000}}
	got := pt.Encode(t, ctx, in.Encode, nil)
	want := []byte{
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime  Encode4 (LE)
		0x03, 0x02, // slot        Encode2 (LE)
		0xD0, 0x29, 0x24, 0x00, // itemId 2370000 = 0x002429D0 Encode4 (LE)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v95 bytes:\n got %x\nwant %x", got, want)
	}
}

// TestSolomonItemUseByteFixtureV72 pins the USE_SOLOMON_ITEM wire against the
// v72 send site (@0x90cb20). No version gate applies: the wire body is
// byte-identical to the v95 fixture above.
func TestSolomonItemUseByteFixtureV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	in := SolomonItemUse{ItemUse: ItemUse{operation: CharacterItemUseSolomonHandle, updateTime: 0x0A0B0C0D, source: 0x0203, itemId: 2370000}}
	got := pt.Encode(t, ctx, in.Encode, nil)
	want := []byte{
		0x0D, 0x0C, 0x0B, 0x0A, // updateTime  Encode4 (LE)
		0x03, 0x02, // slot        Encode2 (LE)
		0xD0, 0x29, 0x24, 0x00, // itemId 2370000 = 0x002429D0 Encode4 (LE)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v72 bytes:\n got %x\nwant %x", got, want)
	}
}
