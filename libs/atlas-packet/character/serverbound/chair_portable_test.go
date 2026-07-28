package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ChairPortable v48 byte-fixture — USE_CHAIR serverbound, op 35.
//
// Client send — CWvsContext::SendSitOnPortableChairRequest @0x712894:
// COutPacket(35)@0x7129c2 then Encode4(itemId)@0x7129da, then
// CUser::SetActivePortableChair(itemId). Single int32 body == v61 ChairPortable
// (portable-chair item path). v48 op 35 (v61 USE_CHAIR=40, Δ-5).
//
// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v48 ida=0x712894
func TestChairPortableV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ChairPortable{itemId: 3010000}.Encode(nil, ctx)(nil)
	want := []byte{0xd0, 0xed, 0x2d, 0x00} // itemId 3010000=0x2DEDD0 (Encode4) /*0x7129da*/
	if !bytes.Equal(got, want) {
		t.Errorf("v48 ChairPortable wire: got %x want %x", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v83 ida=0xa0f9e2
// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v87 ida=0xa9ef5b
// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v95 ida=0x9da100
// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v84 ida=0xa59e1f
// packet-audit:verify packet=character/serverbound/ChairPortable version=jms_v185 ida=0xaf3ee8
func TestChairPortableRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChairPortable{itemId: 3010000}
			output := ChairPortable{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
