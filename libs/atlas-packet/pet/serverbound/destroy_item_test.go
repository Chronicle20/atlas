package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// The DESTROY_PET_ITEM_REQUEST arm of CWvsContext::SendActivatePetRequest. Body
// is tick(4) + liCashItemSN(8), with no version gate: every send site in the
// range encodes Encode4(get_update_time()) then EncodeBuffer(&liCashItemSN, 8).
//
//	gms_v83  fn 0xa240a2  COutPacket(0x50) Encode4 EncodeBuffer(8)
//	gms_v87  fn 0xabbb70  COutPacket(0x53) Encode4@0xabbc7f EncodeBuffer@0xabbc90
//	gms_v92  fn 0x9cb540  COutPacket@0x9cb6db Encode4@0x9cb6f2 EncodeBuffer@0x9cb701
//	gms_v95  fn 0x9f6980  COutPacket(86) Encode4 EncodeBuffer(&v9->liCashItemSN, 8)
//	jms_v185 fn 0xb0b40b  COutPacket@0xb0b50f Encode4@0xb0b524 EncodeBuffer@0xb0b532
//
// v48/v61/v72/v79 have no opcode for this op (registry: n-a) and are not routed.
//
// gms_v83 and gms_v84 carry NO verify marker: their send site is UNNAMED in
// both IDBs (v83 sub_A240A2), so it is absent from the checked-in export and
// `evidence pin` cannot resolve a citation for it. The bytes are pinned below
// for both versions all the same; promoting those two matrix cells needs the
// function named and the column's export re-harvested (RE_AUDITING_A_COLUMN).
//
// packet-audit:verify packet=pet/serverbound/PetDestroyItem version=gms_v87 ida=0xabbb70
// packet-audit:verify packet=pet/serverbound/PetDestroyItem version=gms_v92 ida=0x9cb540
// packet-audit:verify packet=pet/serverbound/PetDestroyItem version=gms_v95 ida=0x9f6980
// packet-audit:verify packet=pet/serverbound/PetDestroyItem version=jms_v185 ida=0xb0b40b
func TestDestroyItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DestroyItem{updateTime: 0x11223344, cashItemSerialNumber: 0x0102030405060708}
			output := DestroyItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.CashItemSerialNumber() != input.CashItemSerialNumber() {
				t.Errorf("cashItemSerialNumber: got %v, want %v", output.CashItemSerialNumber(), input.CashItemSerialNumber())
			}
		})
	}
}

// TestDestroyItemBytes pins the wire against the client's own send order. The
// serial is written by EncodeBuffer over the raw 8-byte liCashItemSN, so it
// lands little-endian — the same bytes Encode8 would produce, which is why the
// codec uses WriteLong.
func TestDestroyItemBytes(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
		minor  uint16
	}{{"GMS", 83, 1}, {"GMS", 84, 1}, {"GMS", 87, 1}, {"GMS", 92, 1}, {"GMS", 95, 1}, {"JMS", 185, 1}} {
		ctx := pt.CreateContext(v.region, v.major, v.minor)
		got := DestroyItem{updateTime: 0x11223344, cashItemSerialNumber: 0x0102030405060708}.Encode(nil, ctx)(nil)
		want := []byte{
			0x44, 0x33, 0x22, 0x11, // tick Encode4 (LE)
			0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // liCashItemSN EncodeBuffer(8) (LE)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s v%d = % X, want % X", v.region, v.major, got, want)
		}
	}
}
