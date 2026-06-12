package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonMoveDecode(t *testing.T) {
	// oid=1000001 (LE 41 42 0F 00), startX=100 (LE 64 00),
	// startY=-50 (LE CE FF), then raw movement blob.
	rawMovement := []byte{0xAA, 0xBB, 0xCC}
	known := []byte{0x41, 0x42, 0x0F, 0x00, 0x64, 0x00, 0xCE, 0xFF}
	known = append(known, rawMovement...)

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			req := request.Request(known)
			reader := request.NewRequestReader(&req, 0)
			var m Move
			m.Decode(l, ctx)(&reader, nil)

			if m.Oid() != 1000001 {
				t.Errorf("oid = %d, want 1000001", m.Oid())
			}
			if m.StartX() != 100 {
				t.Errorf("startX = %d, want 100", m.StartX())
			}
			if m.StartY() != -50 {
				t.Errorf("startY = %d, want -50", m.StartY())
			}
			if !bytes.Equal(m.RawMovement(), rawMovement) {
				t.Errorf("rawMovement = %v, want %v", m.RawMovement(), rawMovement)
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

			in := Move{oid: 1000001, startX: 100, startY: -50, rawMovement: []byte{0x01, 0x02, 0x03}}
			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out Move
			out.Decode(l, ctx)(&reader, nil)
			if out.Oid() != in.oid || out.StartX() != in.startX || out.StartY() != in.startY {
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
