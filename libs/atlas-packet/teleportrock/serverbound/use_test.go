package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Layout is version-invariant: short slot, int itemId, target payload,
// trailing int updateTime (design §1 Q1 — no leading updateTime even on v95).
//
// packet-audit:verify packet=teleportrock/serverbound/Use version=gms_v83 ida=0xa0a3bb
func TestUseByMapDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x02, 0x00, // slot = 2
		0x80, 0x66, 0x23, 0x00, // itemId = 2320000
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Valid() {
		t.Fatalf("expected valid decode")
	}
	if p.Slot() != 2 || p.ItemId() != 2320000 || p.UpdateTime() != 42 {
		t.Fatalf("fields: %+v", p)
	}
	if p.Target().ByName() || p.Target().TargetMap() != 100000000 {
		t.Fatalf("target: %+v", p.Target())
	}
}

// task-124 v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::SendMapTransferItemUseRequest @0x9e6020 — byte-identical read
// order to v83: Encode2(nPOS) @0x9e60c4, Encode4(nItemID) @0x9e60ce, then
// RunMapTransferItem(this, &oPacket, 0) @0x9e60dc; on success
// Encode4(get_update_time()) @0x9e60ef (a genuine TRAILING updateTime — this
// op has no leading-header updateTime on ANY version, unlike the cash
// ItemUseTeleportRock sub-body). Confirms the "version-invariant" claim above
// for v95.
//
// packet-audit:verify packet=teleportrock/serverbound/Use version=gms_v95 ida=0x9e6020
func TestUseByNameDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x01, 0x00, // slot = 1
		0x40, 0xEA, 0x4C, 0x00, // itemId = 5040000
		0x01,       // byName = 1
		0x05, 0x00, // name length = 5
		'A', 'd', 'e', 'l', 'e',
		0x00, 0x00, 0x00, 0x00, // updateTime = 0
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Valid() || !p.Target().ByName() || p.Target().TargetName() != "Adele" {
		t.Fatalf("decode: %+v target %+v", p, p.Target())
	}
}

// task-124 v84 verify pass (live GMS_v84.1_U_DEVM.exe, port 13345):
// CWvsContext::SendMapTransferItemUseRequest @0xa547ab — unnamed in the v84
// IDB (sub_A547AB) until this pass; renamed live to the demangled registry
// fname. Byte-identical read order to v83: guard itemId/10000==232,
// Encode2(nPOS) @0xa547fb, Encode4(nItemID) @0xa54806, then
// CWvsContext::RunMapTransferItem(this, &v11, 0) @0xa54813 (the shared
// target-payload helper, also renamed live from sub_A5489A); on success
// Encode4(get_update_time()) @0xa54825 then SendPacket @0xa54834 — a genuine
// trailing updateTime. Confirms the "version-invariant" claim above for v84.
//
// packet-audit:verify packet=teleportrock/serverbound/Use version=gms_v84 ida=0xa547ab
func TestUseByMapDecodeV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
	b := []byte{
		0x02, 0x00, // slot = 2
		0x80, 0x66, 0x23, 0x00, // itemId = 2320000
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Valid() {
		t.Fatalf("expected valid decode")
	}
	if p.Slot() != 2 || p.ItemId() != 2320000 || p.UpdateTime() != 42 {
		t.Fatalf("fields: %+v", p)
	}
	if p.Target().ByName() || p.Target().TargetMap() != 100000000 {
		t.Fatalf("target: %+v", p.Target())
	}
}

// Client sent the packet with no target payload (dialog closed without a
// selection) — must decode as invalid, never panic.
func TestUseAbsentTargetIsInvalid(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x02, 0x00,
		0x80, 0x66, 0x23, 0x00,
		0x2A, 0x00, 0x00, 0x00, // only updateTime remains
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if p.Valid() {
		t.Fatalf("absent target payload must be invalid")
	}
}

func TestUseRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			in := NewUse(3, 5041000, teleportrock.NewTargetByMap(220000000), 7)
			enc := in.Encode(l, ctx)(nil)
			req := request.Request(enc)
			r := request.NewRequestReader(&req, 0)
			out := Use{}
			out.Decode(l, ctx)(&r, nil)
			if !out.Valid() || out.Slot() != 3 || out.ItemId() != 5041000 ||
				out.Target().TargetMap() != 220000000 || out.UpdateTime() != 7 {
				t.Fatalf("round trip: %+v", out)
			}
		})
	}
}
