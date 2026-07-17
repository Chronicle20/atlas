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

// task-124 v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::SendMapTransferRequest @0x9f3b90 — byte-identical read order to
// v83: Encode1(nType) @0x9f3bd0, Encode1(bCanTransferContinent) @0x9f3bde,
// then `if (!nType) Encode4(dwTargetField)` @0x9f3be5-f0. Confirms the
// "version-invariant" claim above for v95.
//
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=gms_v95 ida=0x9f3b90
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
