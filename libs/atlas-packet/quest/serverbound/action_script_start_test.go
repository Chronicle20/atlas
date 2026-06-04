package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestActionScriptStartRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionScriptStart{npcId: 9001, x: 50, y: 75}
			output := ActionScriptStart{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NpcId() != input.NpcId() {
				t.Errorf("npcId: got %v, want %v", output.NpcId(), input.NpcId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}

// TestActionScriptStartWireShape verifies the wire layout against
// CQuest::StartQuest (GMS v95 @ 0x6b40a0), action=4 (IsStartScriptLinkedQuest branch):
//
//	Encode4 → npcId uint32 LE
//	Encode2 → x int16 LE
//	Encode2 → y int16 LE
//
// All versions identical — 8 bytes total.
func TestActionScriptStartWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ActionScriptStart{npcId: 9001, x: 50, y: 75}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 8 {
				t.Fatalf("wire size = %d bytes, want 8: % x", len(b), b)
			}
			gotNpc := binary.LittleEndian.Uint32(b[0:4])
			if gotNpc != 9001 {
				t.Errorf("npcId = %d, want 9001", gotNpc)
			}
			gotX := int16(binary.LittleEndian.Uint16(b[4:6]))
			if gotX != 50 {
				t.Errorf("x = %d, want 50", gotX)
			}
			gotY := int16(binary.LittleEndian.Uint16(b[6:8]))
			if gotY != 75 {
				t.Errorf("y = %d, want 75", gotY)
			}
		})
	}
}
