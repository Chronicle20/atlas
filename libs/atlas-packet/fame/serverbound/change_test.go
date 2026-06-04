package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{targetId: 12345, mode: 1}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// TestChangeFameWireShape proves the exact wire layout matches what
// CWvsContext::SendGivePopularityRequest (GMS v95 @ 0x9f67e0) encodes:
//
//	Encode4 (dwCharacterId as uint32 LE)
//	Encode1 (bInc as byte)
//
// All versions share this layout — no version gate needed.
func TestChangeFameWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := Change{targetId: 99999, mode: 1}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 4 (targetId uint32) + 1 (mode int8) = 5 bytes
			if len(b) != 5 {
				t.Fatalf("wire size = %d bytes, want 5: % x", len(b), b)
			}
			gotId := binary.LittleEndian.Uint32(b[0:4])
			if gotId != 99999 {
				t.Errorf("targetId = %d, want 99999", gotId)
			}
			if int8(b[4]) != 1 {
				t.Errorf("mode = %d, want 1", int8(b[4]))
			}
		})
	}
}
