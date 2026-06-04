package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestActionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Action{action: 1, questId: 1234}
			output := Action{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ActionType() != input.ActionType() {
				t.Errorf("action: got %v, want %v", output.ActionType(), input.ActionType())
			}
			if output.QuestId() != input.QuestId() {
				t.Errorf("questId: got %v, want %v", output.QuestId(), input.QuestId())
			}
		})
	}
}

// TestActionWireShape verifies the wire layout against
// CWvsContext::ResignQuest (GMS v95 @ 0x9f3cf0), action=3:
//
//	Encode1 → action byte
//	Encode2 → questId uint16 LE
//
// All versions identical — 3 bytes total.
func TestActionWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := Action{action: 3, questId: 5000}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 3 {
				t.Fatalf("wire size = %d bytes, want 3: % x", len(b), b)
			}
			if b[0] != 3 {
				t.Errorf("action = %d, want 3", b[0])
			}
			got := binary.LittleEndian.Uint16(b[1:3])
			if got != 5000 {
				t.Errorf("questId = %d, want 5000", got)
			}
		})
	}
}
