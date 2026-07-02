package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v95 ida=0x9d6260
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v87 ida=0xa9e9ff
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v83 ida=0xa09221
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=jms_v185 ida=0xaeddcb
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v84 ida=0xa53535
//
// v79 (USE_UPGRADE_SCROLL op 84, unnamed twin sub_954F9B @0x954F9B):
// COutPacket(84) + Encode4(get_update_time) + Encode2(scrollSlot) +
// Encode2(equipSlot) + Encode2(bWhiteScroll) + Encode1(legendarySpirit) —
// matches Decode4+Decode2×3+Decode1. Export entry resolved from the unnamed
// twin's decompile.
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v79 ida=0x954f9b
//
// v72 (USE_UPGRADE_SCROLL op 85, sub_903C75 @0x903c75): COutPacket(85) +
// Encode4(updateTime)@0x903cb0 + Encode2(scrollSlot=a2)@0x903cbb + Encode2(equipSlot=a3)
// @0x903cc6 + Encode2(bWhiteScroll=a4)@0x903cd1 + Encode1(legendarySpirit=a5)@0x903cdc —
// identical to v79. No version gate on the codec.
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v72 ida=0x903c75
func TestScrollUseBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := pt.Encode(t, ctx, ScrollUse{updateTime: 12345, scrollSlot: 3, equipSlot: -5, bWhiteScroll: 2, legendarySpirit: true}.Encode, nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime=12345 (LE)
		0x03, 0x00, // scrollSlot=3
		0xFB, 0xFF, // equipSlot=-5
		0x02, 0x00, // bWhiteScroll=2
		0x01, // legendarySpirit=true
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// v61 (USE_UPGRADE_SCROLL op 79, sub_8317A4 @0x8317a4): COutPacket(79) +
// Encode4(updateTime)@0x8317e1 + Encode2(scrollSlot=a2)@0x8317ec + Encode2(equipSlot=a3)
// @0x8317f7 + Encode2(bWhiteScroll=a4)@0x831802 + Encode1(legendarySpirit=a5)@0x83180d —
// body byte-identical to v72. v72 op85 (Δ-6). No version gate on the codec.
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v61 ida=0x8317a4
func TestScrollUseBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := pt.Encode(t, ctx, ScrollUse{updateTime: 12345, scrollSlot: 3, equipSlot: -5, bWhiteScroll: 2, legendarySpirit: true}.Encode, nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime=12345 (LE)
		0x03, 0x00, // scrollSlot=3
		0xFB, 0xFF, // equipSlot=-5
		0x02, 0x00, // bWhiteScroll=2
		0x01, // legendarySpirit=true
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

func TestScrollUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ScrollUse{updateTime: 12345, scrollSlot: 3, equipSlot: -5, bWhiteScroll: 2, legendarySpirit: true}
			output := ScrollUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.ScrollSlot() != input.ScrollSlot() {
				t.Errorf("scrollSlot: got %v, want %v", output.ScrollSlot(), input.ScrollSlot())
			}
			if output.EquipSlot() != input.EquipSlot() {
				t.Errorf("equipSlot: got %v, want %v", output.EquipSlot(), input.EquipSlot())
			}
			if output.WhiteScroll() != input.WhiteScroll() {
				t.Errorf("whiteScroll: got %v, want %v", output.WhiteScroll(), input.WhiteScroll())
			}
			if output.LegendarySpirit() != input.LegendarySpirit() {
				t.Errorf("legendarySpirit: got %v, want %v", output.LegendarySpirit(), input.LegendarySpirit())
			}
		})
	}
}
