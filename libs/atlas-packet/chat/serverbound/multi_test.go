package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestMultiRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Multi{chatType: 1, recipients: []uint32{100, 200, 300}, chatText: "party chat"}
			output := Multi{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChatType() != input.ChatType() {
				t.Errorf("chatType: got %v, want %v", output.ChatType(), input.ChatType())
			}
			if len(output.Recipients()) != len(input.Recipients()) {
				t.Fatalf("recipients length: got %v, want %v", len(output.Recipients()), len(input.Recipients()))
			}
			for i, r := range output.Recipients() {
				if r != input.Recipients()[i] {
					t.Errorf("recipients[%d]: got %v, want %v", i, r, input.Recipients()[i])
				}
			}
			if output.ChatText() != input.ChatText() {
				t.Errorf("chatText: got %v, want %v", output.ChatText(), input.ChatText())
			}
		})
	}
}

func TestMultiUpdateTimeGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := Multi{updateTime: 0x11223344, chatType: 1, recipients: []uint32{7}, chatText: "hi"}
	// GMS v95: leading 4-byte updateTime little-endian.
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if !bytes.Equal(b95[:4], []byte{0x44, 0x33, 0x22, 0x11}) {
		t.Errorf("v95 leading updateTime = % x, want 44 33 22 11", b95[:4])
	}
	// GMS v87: NO updateTime → first byte is chatType.
	b87 := in.Encode(l, pt.CreateContext("GMS", 87, 1))(nil)
	if b87[0] != 0x01 {
		t.Errorf("v87 first byte = 0x%02x, want chatType 0x01", b87[0])
	}
	// GMS v83: NO updateTime.
	b83 := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if b83[0] != 0x01 {
		t.Errorf("v83 first byte = 0x%02x, want chatType 0x01", b83[0])
	}
	// JMS185: NO updateTime.
	bj := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	if bj[0] != 0x01 {
		t.Errorf("JMS first byte = 0x%02x, want chatType 0x01", bj[0])
	}
	// Round-trip every variant.
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		out := Multi{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	}
}
