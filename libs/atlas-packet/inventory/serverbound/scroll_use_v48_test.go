package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestScrollUseBytesV48 — CWvsContext::SendUpgradeItemUseRequest @0x70da60 builds
// COutPacket(66) and encodes Encode4(tick), Encode2(scrollSlot),
// Encode2(equipSlot), Encode2(bWhiteScroll), Encode1(legendarySpirit).
// Shape-stable; the codec already matched.
//
// packet-audit:verify packet=inventory/serverbound/InventoryScrollUse version=gms_v48 ida=0x70da60
func TestScrollUseBytesV48(t *testing.T) {
	in := ScrollUse{updateTime: 0x01020304, scrollSlot: 2, equipSlot: -1, bWhiteScroll: 0, legendarySpirit: false}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime      — Encode4
		0x02, 0x00, // scrollSlot      — Encode2
		0xFF, 0xFF, // equipSlot -1    — Encode2
		0x00, 0x00, // bWhiteScroll    — Encode2
		0x00, // legendarySpirit — Encode1
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 scroll use:\n got % x\nwant % x", got, want)
	}
}
