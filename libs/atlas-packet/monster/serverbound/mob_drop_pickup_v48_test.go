package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMobDropPickupRequestBytesV48 — CMob::SendDropPickUpRequest @0x55220a builds
// COutPacket(131) and encodes Encode4(mobCrc) then Encode4(dropId).
// Shape-stable; the codec already matched.
//
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v48 ida=0x55220a
func TestMobDropPickupRequestBytesV48(t *testing.T) {
	in := MobDropPickupRequest{mobCrc: 0x01020304, dropId: 0x0A0B0C0D}
	got := in.Encode(nil, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // mobCrc — Encode4
		0x0D, 0x0C, 0x0B, 0x0A, // dropId — Encode4
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 mob drop pickup:\n got % x\nwant % x", got, want)
	}
}
