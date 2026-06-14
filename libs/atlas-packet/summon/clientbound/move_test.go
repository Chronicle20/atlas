package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
			// oid only round-trips on v95+ (gated); pre-95 wire carries no oid.
			te := tenant.MustFromContext(ctx)
			if te.IsRegion("GMS") && te.MajorAtLeast(95) {
				if out.Oid() != 1000001 {
					t.Errorf("oid = %d, want 1000001", out.Oid())
				}
			} else if out.Oid() != 0 {
				t.Errorf("pre-95 oid = %d, want 0 (no oid on wire)", out.Oid())
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

// TestSummonMoveBytes pins the classic (pre-95) wire: NO oid (the summon pool is
// cid-keyed on v83/v87; oid is a v95+ addition — IDB-confirmed, summon-wire-truth.md).
// v87 (CSummonedPool::OnMove@0x7f902b -> CMovePath::OnMovePacket@0x6c802d) reads
// the same cid + movement-blob shape as v83 — byte-identical, no oid.
// v84 (CSummonedPool::OnMove sub_7CC317@0x7cc317 -> CMovePath__OnMovePacket@0x6a203f)
// reads the same cid + movement-blob shape — GMS_v84.1 IDB-confirmed, no oid.
// jms185 (CSummonedPool::OnMove@0x8286e4 -> CMovePath::OnMovePacket@0x70c5dc) reads
// the same cid + movement-blob shape — jms185 IDB-confirmed, no oid (the v95+ oid
// is GMS-only; jms185 keeps the pool cid-keyed). The TestSummonMove variant loop
// covers JMS v185 bytes.
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v83 ida=0x7a6861
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v87 ida=0x7f902b
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v84 ida=0x7cc317
// packet-audit:verify packet=summon/clientbound/SummonMove version=jms_v185 ida=0x8286e4
func TestSummonMoveBytes(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, 100, -50, raw)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, NO oid, startX=100=0x0064, startY=-50=0xFFCE, then raw blob
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
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
