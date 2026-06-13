package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestSummonMoveDecode decodes a real-shaped MOVE_SUMMON send: Encode4 summonId
// followed by the opaque CMovePath::Flush blob (whose first 4 bytes are
// startX/startY). Confirmed against CVecCtrlSummoned::EndUpdateActive
// (v83 sub_9C84E9, v87 @0xa591da, v95 @0x9a0700). v87 body is byte-identical to
// v83 (Encode4 summonId=ctrl[188]=cid + opaque CMovePath::Flush blob).
// packet-audit:verify packet=summon/serverbound/SummonMoveHandle version=gms_v95 ida=0x9a0700
// packet-audit:verify packet=summon/serverbound/SummonMoveHandle version=gms_v83 ida=0x9c84e9
// packet-audit:verify packet=summon/serverbound/SummonMoveHandle version=gms_v87 ida=0xa591da
func TestSummonMoveDecode(t *testing.T) {
	// summonId=1000001 (LE 41 42 0F 00), then the move blob: startX=100 (LE 64 00),
	// startY=-50 (LE CE FF), then the remaining (opaque) move-path bytes.
	moveBlob := []byte{0x64, 0x00, 0xCE, 0xFF, 0xAA, 0xBB, 0xCC}
	known := []byte{0x41, 0x42, 0x0F, 0x00}
	known = append(known, moveBlob...)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			req := request.Request(known)
			reader := request.NewRequestReader(&req, 0)
			var m Move
			m.Decode(l, ctx)(&reader, nil)

			if m.SummonId() != 1000001 {
				t.Errorf("summonId = %d, want 1000001", m.SummonId())
			}
			if m.StartX() != 100 {
				t.Errorf("startX = %d, want 100", m.StartX())
			}
			if m.StartY() != -50 {
				t.Errorf("startY = %d, want -50", m.StartY())
			}
			if !bytes.Equal(m.RawMovement(), moveBlob) {
				t.Errorf("rawMovement = %v, want %v", m.RawMovement(), moveBlob)
			}
			if reader.Available() > 0 {
				t.Errorf("reader has %d unconsumed bytes", reader.Available())
			}
		})
	}
}

func TestSummonMoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			// rawMovement begins with startX/startY (the first 4 bytes of the blob).
			in := Move{summonId: 1000001, startX: 100, startY: -50, rawMovement: []byte{0x64, 0x00, 0xCE, 0xFF, 0x01, 0x02, 0x03}}
			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out Move
			out.Decode(l, ctx)(&reader, nil)
			if out.SummonId() != in.summonId || out.StartX() != in.startX || out.StartY() != in.startY {
				t.Errorf("round-trip mismatch: got %+v", out)
			}
			if !bytes.Equal(out.RawMovement(), in.rawMovement) {
				t.Errorf("rawMovement mismatch: got %v want %v", out.RawMovement(), in.rawMovement)
			}
			if reader.Available() > 0 {
				t.Errorf("reader has %d unconsumed bytes", reader.Available())
			}
		})
	}
}
