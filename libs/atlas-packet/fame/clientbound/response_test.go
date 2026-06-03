package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestReceiveFameResponse(t *testing.T) {
	input := NewReceiveFameResponse(0, "Player1", 1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestGiveFameResponse(t *testing.T) {
	input := NewGiveFameResponse(1, "Player2", 1, 50)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFameErrorResponse(t *testing.T) {
	input := NewFameErrorResponse(5)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestReceiveFameResponseWireShape proves the exact wire layout for case 5
// (RECEIVE) of CWvsContext::OnGivePopularityResult (GMS v95 @ 0x9fea60):
//
//	Decode1  (mode)
//	DecodeStr(fromName)     — 2-byte LE length prefix + ShiftJIS bytes
//	Decode1  (inc/dec flag)
//
// All versions share this layout — no version gate needed.
func TestReceiveFameResponseWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// mode=5 (RECEIVE), fromName="P1" (2 ASCII bytes), amount=+1 → fameMode=(1+1)/2=1
	in := NewReceiveFameResponse(5, "P1", 1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode) + 2 (len prefix) + 2 (name bytes) + 1 (inc/dec byte) = 6
			if len(b) != 6 {
				t.Fatalf("wire size = %d bytes, want 6: % x", len(b), b)
			}
			if b[0] != 0x05 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x05", b[0])
			}
			nameLen := int(binary.LittleEndian.Uint16(b[1:3]))
			if nameLen != 2 {
				t.Errorf("name length prefix = %d, want 2", nameLen)
			}
			if b[3] != 'P' || b[4] != '1' {
				t.Errorf("name bytes = % x, want [50 31]", b[3:5])
			}
			if b[5] != 0x01 {
				t.Errorf("inc/dec byte = 0x%02x, want 0x01 (inc)", b[5])
			}
		})
	}
}

// TestGiveFameResponseWireShape proves the exact wire layout for case 0
// (GIVE) of CWvsContext::OnGivePopularityResult (GMS v95 @ 0x9fea60):
//
//	Decode1  (mode)
//	DecodeStr(toName)       — 2-byte LE length prefix + ShiftJIS bytes
//	Decode1  (inc/dec flag)
//	Decode4  (new fame total as int32 LE)
//
// Atlas encodes total as WriteInt16 + WriteShort(0) which produces the
// same 4 wire bytes as Decode4(int32) for values in the int16 range.
// All versions share this layout.
func TestGiveFameResponseWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// mode=0 (GIVE), toName="P2" (2 ASCII bytes), amount=+1, total=50
	in := NewGiveFameResponse(0, "P2", 1, 50)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode) + 2 (len prefix) + 2 (name bytes) + 1 (inc/dec byte) + 4 (total) = 10
			if len(b) != 10 {
				t.Fatalf("wire size = %d bytes, want 10: % x", len(b), b)
			}
			if b[0] != 0x00 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x00", b[0])
			}
			nameLen := int(binary.LittleEndian.Uint16(b[1:3]))
			if nameLen != 2 {
				t.Errorf("name length prefix = %d, want 2", nameLen)
			}
			if b[3] != 'P' || b[4] != '2' {
				t.Errorf("name bytes = % x, want [50 32]", b[3:5])
			}
			if b[5] != 0x01 {
				t.Errorf("inc/dec byte = 0x%02x, want 0x01 (inc)", b[5])
			}
			// total=50 as 4-byte LE int32
			total := int32(binary.LittleEndian.Uint32(b[6:10]))
			if total != 50 {
				t.Errorf("total = %d, want 50 (0x%08x)", total, uint32(total))
			}
		})
	}
}

// TestFameErrorResponseWireShape proves the exact wire layout for error
// cases of CWvsContext::OnGivePopularityResult (GMS v95 @ 0x9fea60):
//
//	Decode1 (mode only — no additional fields)
//
// All versions share this single-byte layout.
func TestFameErrorResponseWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewFameErrorResponse(3)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 1 {
				t.Errorf("wire size = %d bytes, want 1: % x", len(b), b)
			}
			if b[0] != 0x03 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x03", b[0])
			}
		})
	}
}
