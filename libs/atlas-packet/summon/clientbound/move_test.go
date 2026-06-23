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
	in := NewSummonMove(42, 1000001, raw)
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
			if !bytes.Equal(out.RawMovement(), raw) {
				t.Errorf("rawMovement = %v, want %v", out.RawMovement(), raw)
			}
		})
	}
}

// TestSummonMoveBytes pins the v83 wire: cid + oid + the raw CMovePath blob, with
// NO separate start position. The blob already begins with start x,y (CMovePath::
// Encode), and the client reads it via CMovePath::Decode (v83 @0x68a33c, from
// CSummonedPool::OnMove@0x7a6861). Writing the position separately mis-aligns the
// observer's decode by 4 bytes and crashes the client (ZException / error 38);
// the owner renders movement locally and never receives this packet, so only
// observers hit it. cid is read upstream by CUserPool::OnUserCommonPacket@0x972401;
// CSummonedPool::OnPacket@0x938dd7 reads the oid before OnMove. NOTE: the summon
// clientbound matrix cells are demoted pending a packet-verifier re-pin against
// the corrected dispatcher (see summon-wire-truth.md).
func TestSummonMoveBytes(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, raw)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, then the raw movement blob (no separate pos)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement (CMovePath blob)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonMoveBytesV83 pins the v83 wire byte-for-byte against the live
// decompile. Dispatch chain (IDA, MapleStory_dump.exe @port 13341):
//   - CUserPool::OnUserCommonPacket@0x972401 reads cid (Decode4@0x97240c), routes
//     op 0xB1 to CSummonedPool::OnPacket@0x972490.
//   - CSummonedPool::OnPacket@0x938dd7 reads oid (Decode4@0x938e16), looks up the
//     summon, then case 0xB1 calls CSummonedPool::OnMove(v9,v5)@0x938ea7.
//   - CSummonedPool::OnMove@0x7a6861 forwards the CInPacket to
//     CMovePath::OnMovePacket@0x68b371, which decodes the raw movement blob (the
//     blob itself begins with start x,y per CMovePath::Encode).
// So the wire is: int cid (consumed upstream) + int oid + raw CMovePath blob, with
// NO separately-written start position (writing one mis-aligns the observer decode).
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v83 ida=0x7a6861
func TestSummonMoveBytesV83(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, raw)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, then the raw movement blob (no separate pos)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x938e16 in OnPacket)
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement (CMovePath blob → OnMovePacket@0x68b371)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 bytes = % X, want % X", got, want)
	}
}

// TestSummonMoveBytesV84 pins the v84 wire byte-for-byte against the live
// decompile (IDA, GMS_v84.1_U_DEVM.exe @port 13337). v84 is v83-shaped — same
// dispatch + leaf, no version delta. Dispatch chain:
//   - CUserPool::OnUserCommonPacket@0x9b23a1 reads cid (Decode4@0x9b23ac), routes
//     op 0xB5 (181) to the summon dispatcher sub_970201@0x970201.
//   - sub_970201@0x970201 reads oid (Decode4@0x970240), looks up the summon, then
//     case 181 calls the OnMove leaf sub_7CC317@0x7cc317.
//   - sub_7CC317@0x7cc317 forwards the CInPacket to CMovePath__OnMovePacket
//     (@0x6a203f), which decodes the raw movement blob (the blob itself begins with
//     start x,y per CMovePath::Encode).
// Wire = int cid (consumed upstream) + int oid + raw CMovePath blob, with NO
// separately-written start position. No off-by-one: Move has no version gate, so the
// v84 path is byte-identical to v83.
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v84 ida=0x7cc317
func TestSummonMoveBytesV84(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, raw)
	ctx := test.CreateContext("GMS", 84, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	// cid=42, oid=1000001=0x000F4241, then the raw movement blob (no separate pos)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid (consumed by dispatcher)
		0x41, 0x42, 0x0F, 0x00, // oid (Decode4@0x970240 in sub_970201)
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement (CMovePath blob → CMovePath__OnMovePacket@0x6a203f)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v84 bytes = % X, want % X", got, want)
	}
}

// TestSummonMoveBytesV95 confirms v95 carries the same shape (cid + oid + blob) —
// there is no v95-specific move delta beyond the (now universal) oid.
// packet-audit:verify packet=summon/clientbound/SummonMove version=gms_v95 ida=0x759830
func TestSummonMoveBytesV95(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	in := NewSummonMove(42, 1000001, raw)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // cid
		0x41, 0x42, 0x0F, 0x00, // oid
		0x01, 0x02, 0x03, 0x04, 0x05, // rawMovement
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
}
