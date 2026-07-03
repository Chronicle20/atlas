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
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v72 ida=0x904088
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v61 ida=0x831bb7
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

// TestItemCancelByteFixtureV72 pins the CANCEL_ITEM_EFFECT (send op 72) wire
// against CWvsContext::SendStatChangeItemCancelRequest (v72 sub_904088 @0x904088).
// After the update-time (sub_4DBE16) and item-category guard the client emits
// COutPacket(72) @0x904114 + Encode4(sourceId) @0x904126. op 72 = v79 op 71 + 1
// (mid/field/social Δ+1); single int32 body matches ItemCancel.Encode.
func TestItemCancelByteFixtureV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	// sourceId=2001001 (0x001E8869 LE)
	got := pt.Encode(t, ctx, ItemCancel{sourceId: 2001001}.Encode, nil)
	want := []byte{0x69, 0x88, 0x1E, 0x00} // sourceId (Encode4) /*0x904126*/
	if !bytes.Equal(got, want) {
		t.Errorf("v72 bytes:\n got %x\nwant %x", got, want)
	}
}

// TestItemCancelByteFixtureV61 pins CANCEL_ITEM_EFFECT (v61 send op 68) against
// CWvsContext::SendStatChangeItemCancelRequest sub_831BB7@0x831bb7: the item-buff
// right-click cancel path (CWndMan proc sub_4486B0 v5==1 -> sub_831BB7(-sourceId))
// builds COutPacket(68) then a single Encode4(sourceId) @0x831c55. No version gate.
func TestItemCancelByteFixtureV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	// sourceId=2001001 (0x001E8869 LE)
	got := pt.Encode(t, ctx, ItemCancel{sourceId: 2001001}.Encode, nil)
	want := []byte{0x69, 0x88, 0x1E, 0x00} // sourceId (Encode4) @0x831c55
	if !bytes.Equal(got, want) {
		t.Errorf("v61 bytes:\n got %x\nwant %x", got, want)
	}
}
