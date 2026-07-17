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

// task-124 v84 verify pass (live GMS_v84.1_U_DEVM.exe, port 13345):
// CWvsContext::SendMapTransferRequest @0xa71972 — unnamed in the v84 IDB
// (sub_A71972) until this pass, renamed live (byte-identical to v83's
// sub_A261BC @0xa261bc, also unnamed there). COutPacket::COutPacket(&v4, 102)
// @0xa71984; Encode1(a1=nType) @0xa71993; Encode1(a3=bCanTransferContinent)
// @0xa7199e; if(!a1) Encode4(a2=dwTargetField) @0xa719af. Callers confirmed:
// sub_865737 (register UI path) calls SendMapTransferRequest(1, 0, vipFlag);
// sub_865A45 (delete UI path) calls SendMapTransferRequest(0, mapId,
// vipFlag). Confirms the "version-invariant" claim above for v84.
//
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=gms_v84 ida=0xa71972
func TestAddMapDeleteDecodeV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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

// task-124 v87 verify pass (live GMSv87_4GB.exe, port 13343):
// CWvsContext::SendMapTransferRequest @0xabde10 — already named in the v87
// IDB. Byte-identical read order to v83/v84/v95: COutPacket::COutPacket(&a3,
// 0x69) @0xabde22; Encode1(nType) @0xabde31; Encode1(bCanTransferContinent)
// @0xabde3c; if(!nType) Encode4(dwTargetField) @0xabde4d; SendPacket
// @0xabde5c. Confirms the "version-invariant" claim above for v87.
//
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=gms_v87 ida=0xabde10
func TestAddMapDeleteDecodeV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 87, 1)
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

// task-124 jms_v185 verify pass (live MapleStory_dump_SCY.exe, port 13344):
// CWvsContext::SendMapTransferRequest @0xb0d9eb — already named in the jms
// IDB. Byte-identical read order to v83/v84/v87/v95: COutPacket::COutPacket(
// &v5,0x61) @0xb0d9fd (opcode 97, matches registry TROCK_ADD_MAP);
// Encode1(nType) @0xb0da0c; Encode1(bCanTransferContinent) @0xb0da17;
// if(!nType) Encode4(dwTargetField) @0xb0da28; SendPacket @0xb0da37. Confirms
// the "version-invariant" claim above for jms_v185.
//
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=jms_v185 ida=0xb0d9eb
func TestAddMapDeleteDecodeJms(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("JMS", 185, 1)
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
