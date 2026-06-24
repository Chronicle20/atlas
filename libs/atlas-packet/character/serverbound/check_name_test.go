package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/CheckName version=gms_v83 ida=0x7d75ab
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v84 ida=0x60cf5d
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v87 ida=0x62f779
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v95 ida=0x5d5690
// packet-audit:verify packet=character/serverbound/CheckName version=jms_v185 ida=0x66e467
func TestCheckNameRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CheckName{name: "TestChar"}
			output := CheckName{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}

// TestCheckNameJMSGolden pins the exact jms_v185 wire for CheckName against
// CLogin::SendCheckDuplicateIDPacket @0x66e467: COutPacket(8) then EncodeStr(s)
// — a single Shift-JIS length-prefixed name, no other fields.
//   EncodeStr("TestChar") = short(8) + "TestChar".
func TestCheckNameJMSGolden(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	got := CheckName{name: "TestChar"}.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x00, // EncodeStr length = 8
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
	}
	if !bytes.Equal(got, want) {
		t.Errorf("jms CheckName wire: got %x want %x", got, want)
	}
}
