package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v83 ida=0x537a6a
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v87 ida=0x55f3e5
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v95 ida=0x539910
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=jms_v185 ida=0x574ca3
//
// v84 is VERSION-ABSENT: CField::OnStalkResult does not exist in the v84 IDB/export
// (the foothold/stalk cluster is version-divergent) — no marker, recorded ⬜.
func TestStalkResultGolden(t *testing.T) {
	// One stalkee, insert branch (flag=0): count=1, charId=0x11223344, flag=0,
	// name="GM", x=100, y=200.
	input := NewStalkResult(1, 0x11223344, 0, "GM", 100, 200)
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // count=1            (Decode4 @0x537a6a)
		0x44, 0x33, 0x22, 0x11, // charId=0x11223344  (Decode4)
		0x00,                   // flag=0 (insert)    (Decode1)
		0x02, 0x00, 'G', 'M',   // name="GM"          (DecodeStr)
		0x64, 0x00, 0x00, 0x00, // x=100              (Decode4)
		0xC8, 0x00, 0x00, 0x00, // y=200              (Decode4)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("golden mismatch:\n got %v\nwant %v", actual, want)
	}
}

func TestStalkResultRoundTrip(t *testing.T) {
	input := NewStalkResult(1, 0x11223344, 0, "GM", 100, 200)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
