package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v83 ida=0xa096af
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v87 ida=0xa9ef5b
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v95 ida=0x9d9dd0
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v84 ida=0xa53a91
// packet-audit:verify packet=character/serverbound/ItemCancel version=jms_v185 ida=0xaee339
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v79 ida=0x9553cd
func TestItemCancelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemCancel{sourceId: 2001001}
			output := ItemCancel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SourceId() != input.SourceId() {
				t.Errorf("sourceId: got %v, want %v", output.SourceId(), input.SourceId())
			}
		})
	}
}

// TestItemCancelByteFixtureV79 pins the CANCEL_ITEM_EFFECT (send op 71) wire
// against CWvsContext::SendStatChangeItemCancelRequest (v79 @0x9553cd,
// byte-signature twin of v83 @0xa096af). After the IsNoCancelMouse guard the
// client emits COutPacket(71) + Encode4(sourceId):
//
//	sourceId = Encode4  /*0x95546b*/
func TestItemCancelByteFixtureV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// sourceId=2001001 (0x001E8869 LE)
	got := pt.Encode(t, ctx, ItemCancel{sourceId: 2001001}.Encode, nil)
	want := []byte{0x69, 0x88, 0x1E, 0x00} // sourceId (Encode4) /*0x95546b*/
	if !bytes.Equal(got, want) {
		t.Errorf("v79 bytes:\n got %x\nwant %x", got, want)
	}
}
