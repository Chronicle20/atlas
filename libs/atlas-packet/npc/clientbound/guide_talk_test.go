package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestGuideTalkMessage exercises the round-trip across all tenant variants and
// pins the leading branch byte. Per the v95 client (CUserLocal::OnTutorMsg
// @0x916f60) the string/message arm is bByMessage==0, so the first wire byte
// MUST be 0x00.
func TestGuideTalkMessage(t *testing.T) {
	input := NewGuideTalkMessage("Hello adventurer!", 200, 4000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)

			l, _ := testlog.NewNullLogger()
			b := input.Encode(l, ctx)(nil)
			if len(b) == 0 || b[0] != 0x00 {
				t.Fatalf("GuideTalkMessage leading byte: got %#v, want 0x00 (message arm)", b)
			}
		})
	}
}

// TestGuideTalkIdx exercises the round-trip across all tenant variants and pins
// the leading branch byte. Per the v95 client (CUserLocal::OnTutorMsg
// @0x916f60) the hint-index arm is bByMessage!=0, so the first wire byte MUST
// be 0x01.
func TestGuideTalkIdx(t *testing.T) {
	input := NewGuideTalkIdx(5, 7000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)

			l, _ := testlog.NewNullLogger()
			b := input.Encode(l, ctx)(nil)
			if len(b) == 0 || b[0] != 0x01 {
				t.Fatalf("GuideTalkIdx leading byte: got %#v, want 0x01 (index arm)", b)
			}
		})
	}
}
