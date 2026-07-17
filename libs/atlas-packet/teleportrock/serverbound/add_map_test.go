package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Layout is version-invariant (design §1 Q1): byte nType, byte
// bCanTransferContinent, then int dwTargetField ONLY when nType==0 (delete).
// On register the client sends no map id — the server uses session state.
//
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=gms_v83 ida=0xa261bc
//
// NOTE: no gms_v95 marker is carried here yet (task-124 is a v83-only verify
// pass) — a marker without a v95 audit report/evidence record would itself
// register as an orphan (mirrors the ItemUseVegaScroll convention in
// cash/serverbound/item_use_vega_scroll_test.go). The 0x9f3b90 address noted
// above is a design-time claim, not IDA-reverified this session; a separate
// gms_v95 verify pass should add its own marker once that report lands.
func TestAddMapRegisterDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{0x01, 0x01} // register, VIP list — nothing else on the wire
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := AddMap{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Register() || !p.Vip() || p.MapId() != 0 {
		t.Fatalf("decode: %+v", p)
	}
}

func TestAddMapDeleteDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x00, 0x00, // delete, regular list
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
	}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	p := AddMap{}
	p.Decode(l, ctx)(&r, nil)
	if p.Register() || p.Vip() || p.MapId() != 100000000 {
		t.Fatalf("decode: %+v", p)
	}
}

func TestAddMapRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			for _, in := range []AddMap{
				NewAddMap(true, false, 0),
				NewAddMap(false, true, 220000000),
			} {
				enc := in.Encode(l, ctx)(nil)
				req := request.Request(enc)
				r := request.NewRequestReader(&req, 0)
				out := AddMap{}
				out.Decode(l, ctx)(&r, nil)
				if out.Register() != in.Register() || out.Vip() != in.Vip() || out.MapId() != in.MapId() {
					t.Fatalf("round trip: in=%+v out=%+v", in, out)
				}
			}
		})
	}
}
