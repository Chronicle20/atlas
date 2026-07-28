package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ChairFixed v48 byte-fixture — CANCEL_CHAIR serverbound, op 34.
//
// Client send — CUserLocal::HandleXKeyDown @0x69df44 (X-key-down handler; the
// a2==39 arm is the get-up-from-chair send): COutPacket(34)@0x69dffe then
// Encode2(0xFFFF)@0x69e00f (seat index; 0xFFFF (-1) = get-up-from-chair) after
// CWvsContext::SetExclRequestSent. Single int16 body == v61 ChairFixed; the
// get-up path always sends 0xFFFF. v48 op 34 (v61 CANCEL_CHAIR=39, Δ-5).
//
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v48 ida=0x69df44
func TestChairFixedV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ChairFixed{chairId: -1}.Encode(nil, ctx)(nil)
	want := []byte{0xff, 0xff} // chairId 0xFFFF (Encode2) /*0x69e00f*/
	if !bytes.Equal(got, want) {
		t.Errorf("v48 ChairFixed wire: got %x want %x", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v83 ida=0x94e45f
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v87 ida=0x9c9270
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v95 ida=0x90f6d0
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v84 ida=0x986138
// packet-audit:verify packet=character/serverbound/ChairFixed version=jms_v185 ida=0xa0e95a
func TestChairFixedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChairFixed{chairId: 42}
			output := ChairFixed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChairId() != input.ChairId() {
				t.Errorf("chairId: got %v, want %v", output.ChairId(), input.ChairId())
			}
		})
	}
}
