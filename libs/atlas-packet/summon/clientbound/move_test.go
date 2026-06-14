package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSummonMove(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, 100, -50, raw)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)

			// Field-level assertions: encode then decode and verify fields.
			l, _ := testlog.NewNullLogger()
			b := in.Encode(l, ctx)(nil)
			req := request.Request(b)
			reader := request.NewRequestReader(&req, 0)
			var out SummonMove
			out.Decode(l, ctx)(&reader, nil)
			if out.Cid() != 42 {
				t.Errorf("cid = %d, want 42", out.Cid())
			}
			// oid round-trips on ALL versions: cid is read upstream by
			// CUserPool::OnUserCommonPacket, so the per-op Decode4 is the oid.
			if out.Oid() != 1000001 {
				t.Errorf("oid = %d, want 1000001", out.Oid())
			}
			if out.StartX() != 100 {
				t.Errorf("startX = %d, want 100", out.StartX())
			}
			if out.StartY() != -50 {
				t.Errorf("startY = %d, want -50", out.StartY())
			}
			if !bytes.Equal(out.RawMovement(), raw) {
				t.Errorf("rawMovement = %v, want %v", out.RawMovement(), raw)
			}
		})
	}
}

// TestSummonMoveBytes pins the v83 wire: cid + oid + startX + startY + movement
// blob. The cid is read upstream by CUserPool::OnUserCommonPacket@0x972401;
// CSummonedPool::OnPacket@0x938dd7 then does one Decode4 = the oid before OnMove.
// (The prior "no oid" reading missed the upstream cid read and labeled the oid as
// the cid — see summon-wire-truth.md.) NOTE: v84/v87/jms inherit this correction
// by the same dispatcher logic; their matrix cells need re-verification against the
// cid-pre-reading dispatcher (the old ida= markers analyzed the wrong layer).
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v83 ida=0x938dd7
func TestSummonMoveBytes(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, 100, -50, raw)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, startX=100=0x0064, startY=-50=0xFFCE, raw blob
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid
		0x64, 0x00, // startX
		0xCE, 0xFF, // startY
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonMoveBytesV95 pins the v95+ DELTA: the oid int after cid.
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v95 ida=0x759830
func TestSummonMoveBytesV95(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, 100, -50, raw)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=0x000F4241, startX=100, startY=-50, raw blob
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid (v95+ only)
		0x64, 0x00, // startX
		0xCE, 0xFF, // startY
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}
