package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v83 ida=0x94fa8e
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v87 ida=0x9c9270
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v95 ida=0x933920
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v84 ida=0x987824
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=jms_v185 ida=0xa10f9c
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v79 ida=0x8a80a3
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v72 ida=0x85d00c
func TestChalkboardCloseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChalkboardClose{}
			output := ChalkboardClose{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestChalkboardCloseByteFixtureV79 pins the CLOSE_CHALKBOARD (send op 48) wire
// against CUserLocal::HandleLButtonClk (v79 @0x8a80a3, byte-signature twin of
// v83 @0x94fa8e). On a close-area click the client emits COutPacket(48) with an
// EMPTY body (no Encode calls @0x8a80db) — matching ChalkboardClose's empty codec.
func TestChalkboardCloseByteFixtureV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, ChalkboardClose{}.Encode, nil)
	if len(got) != 0 {
		t.Errorf("expected empty body, got % x", got)
	}
}

// TestChalkboardCloseByteFixtureV72 pins the CLOSE_CHALKBOARD (send op 49) wire.
// The v72 close-send lives in the WM_LBUTTONUP arm (case 514) of the mouse
// dispatcher sub_85AB5B, which calls sub_85D00C @0x85d00c: on a close-area hit
// (sub_47F32A) it emits COutPacket(49) @0x85d044 then SendPacket immediately —
// NO Encode calls, an empty body matching ChalkboardClose's empty codec.
// op 49 = v79 op 48 + 1 (mid/field/social Δ+1).
func TestChalkboardCloseByteFixtureV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := pt.Encode(t, ctx, ChalkboardClose{}.Encode, nil)
	if len(got) != 0 {
		t.Errorf("expected empty body, got % x", got)
	}
}
