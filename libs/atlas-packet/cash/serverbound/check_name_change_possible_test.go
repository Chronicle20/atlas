package serverbound

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// ---------------------------------------------------------------------------
// NAME_TRANSFER (serverbound) — CCashShop::SendCheckNameChangePossiblePacket.
//
// Nine applicable cells. jms_v185 gets NO marker: the op is absent from that
// client (derivation.md §1.5 — no 5400000 arm in CCashShop::ProcessBuy
// @0x484ca8, zero hits for the immediate 5400000, and none of the name-change
// UI classes exist). Its serverbound 0x009 is WORLD_TRANSFER.
//
// The `ida=` address on each marker is that version's send site, read out of
// the version's own IDB (derivation.md §1.2) and re-confirmed by decompiling
// each one during this task: every GMS version builds COutPacket(16) except
// gms_v48, which builds COutPacket(18).
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v48 ida=0x44f93e
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v61 ida=0x45cd72
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v72 ida=0x46bc47
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v79 ida=0x46cdbe
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v83 ida=0x4733ca
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v84 ida=0x475f14
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v87 ida=0x47dd48
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v92 ida=0x47f830
// packet-audit:verify packet=cash/serverbound/CashCheckNameChangePossible version=gms_v95 ida=0x488190
func TestCheckNameChangePossibleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckNameChangePossible(123456, 19900102, "19900102")
			output := CheckNameChangePossible{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %d, want %d", output.CharacterId(), input.CharacterId())
			}
			if credentialIsString(ctx) {
				if output.Spw() != input.Spw() {
					t.Errorf("spw: got %q, want %q", output.Spw(), input.Spw())
				}
				if output.BirthDate() != 0 {
					t.Errorf("birthDate: got %d, want 0 (this version sends the string form)", output.BirthDate())
				}
			} else {
				if output.BirthDate() != input.BirthDate() {
					t.Errorf("birthDate: got %d, want %d", output.BirthDate(), input.BirthDate())
				}
				if output.Spw() != "" {
					t.Errorf("spw: got %q, want empty (this version sends the integer form)", output.Spw())
				}
			}
		})
	}
}

// TestCheckNameChangePossibleStringRedactsCredential — the second field is the
// account second password / birthday code, and every serverbound handler in
// atlas-channel logs p.String() at debug level. Neither form of the credential
// may appear in that line, in whole or in part.
func TestCheckNameChangePossibleStringRedactsCredential(t *testing.T) {
	m := NewCheckNameChangePossible(777, 19771231, "19771231")
	s := m.String()
	for _, secret := range []string{"19771231", "1977", "1231"} {
		if strings.Contains(s, secret) {
			t.Errorf("String() leaked the credential (%q): %s", secret, s)
		}
	}
	if !strings.Contains(s, "REDACTED") {
		t.Errorf("String() = %q, want it to say REDACTED", s)
	}
	if !strings.Contains(s, "777") {
		t.Errorf("String() = %q, want it to report the characterId", s)
	}
}

func TestCheckNameChangePossibleOperation(t *testing.T) {
	if (CheckNameChangePossible{}).Operation() != CashShopCheckNameChangePossibleHandle {
		t.Errorf("Operation() = %q, want %q", (CheckNameChangePossible{}).Operation(), CashShopCheckNameChangePossibleHandle)
	}
	if CashShopCheckNameChangePossibleHandle != "CashShopCheckNameChangePossibleHandle" {
		t.Errorf("CashShopCheckNameChangePossibleHandle = %q", CashShopCheckNameChangePossibleHandle)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// The round-trip test is SYMMETRIC: a change to the shared framing moves
// encode and decode together and stays green while every real client request
// desyncs. The fixtures below pin the exact bytes per version.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteInt: binary.LittleEndian uint32.
//   - WriteAsciiString: WriteShort(len) little-endian, then the Shift-JIS bytes
//     (ASCII-identical for the digits used here).
//   - libs/atlas-socket/request/reader.go mirrors both.
//
// Field order per version: derivation.md §2.1, re-confirmed by decompiling each
// send site (two Encode4 on gms_v48…gms_v92; Encode4 + EncodeStr on gms_v95).
// ---------------------------------------------------------------------------

type nameTransferFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
	// stringCredential: the version encodes the credential with EncodeStr
	// (gms_v95 only); every earlier version uses Encode4.
	stringCredential bool
}

// The nine cells this task claims. jms_v185 is deliberately absent — the op
// does not exist in that client.
var nameTransferFixtureVersions = []nameTransferFixtureVersion{
	{"gms_v48", "GMS", 48, 1, false},
	{"gms_v61", "GMS", 61, 1, false},
	{"gms_v72", "GMS", 72, 1, false},
	{"gms_v79", "GMS", 79, 1, false},
	{"gms_v83", "GMS", 83, 1, false},
	{"gms_v84", "GMS", 84, 1, false},
	{"gms_v87", "GMS", 87, 1, false},
	{"gms_v92", "GMS", 92, 1, false},
	{"gms_v95", "GMS", 95, 1, true},
}

// Fixture values and their hand-derived wire bytes.
//
//	characterId 0x0001E240 (123456)  -> 40 E2 01 00
//	birthDate   0x012FA6C6 (19900102) -> C6 A6 2F 01
//	spw         "19900102"            -> 08 00 31 39 39 30 30 31 30 32
const (
	fixtureCharacterId = uint32(123456)
	fixtureBirthDate   = uint32(19900102)
	fixtureSpw         = "19900102"
)

var (
	wireCharacterId = []byte{0x40, 0xE2, 0x01, 0x00}
	wireBirthDate   = []byte{0xC6, 0xA6, 0x2F, 0x01}
	wireSpw         = []byte{0x08, 0x00, 0x31, 0x39, 0x39, 0x30, 0x30, 0x31, 0x30, 0x32}
)

func decodeNameTransferBytes(t *testing.T, v nameTransferFixtureVersion, body []byte) CheckNameChangePossible {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(v.region, v.major, v.minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CheckNameChangePossible
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", v.name, reader.Available(), len(body))
	}
	return out
}

func TestCheckNameChangePossibleByteFixture(t *testing.T) {
	for _, v := range nameTransferFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := append([]byte{}, wireCharacterId...)
			if v.stringCredential {
				want = append(want, wireSpw...)
			} else {
				want = append(want, wireBirthDate...)
			}

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCheckNameChangePossible(fixtureCharacterId, fixtureBirthDate, fixtureSpw)
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}

			out := decodeNameTransferBytes(t, v, want)
			if out.CharacterId() != fixtureCharacterId {
				t.Errorf("%s characterId: got %d, want %d", v.name, out.CharacterId(), fixtureCharacterId)
			}
			if v.stringCredential {
				if out.Spw() != fixtureSpw {
					t.Errorf("%s spw: got %q, want %q", v.name, out.Spw(), fixtureSpw)
				}
			} else if out.BirthDate() != fixtureBirthDate {
				t.Errorf("%s birthDate: got %d, want %d", v.name, out.BirthDate(), fixtureBirthDate)
			}
		})
	}
}

// TestCheckNameChangePossibleV95ChangesWireType pins the v95 boundary
// explicitly: the second field is not merely widened at v95, it changes wire
// type from Encode4 to EncodeStr. A gate written as `> 83` (rather than
// MajorAtLeast(95)) would route v84 — and everything up to v92 — down the
// string path and desync every request.
func TestCheckNameChangePossibleV95ChangesWireType(t *testing.T) {
	m := NewCheckNameChangePossible(fixtureCharacterId, fixtureBirthDate, fixtureSpw)

	v92 := pt.Encode(t, pt.CreateContext("GMS", 92, 1), m.Encode, nil)
	v95 := pt.Encode(t, pt.CreateContext("GMS", 95, 1), m.Encode, nil)
	v84 := pt.Encode(t, pt.CreateContext("GMS", 84, 1), m.Encode, nil)
	v83 := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)

	if len(v92) != 8 {
		t.Errorf("gms_v92 body = %d bytes, want 8 (Encode4 + Encode4)", len(v92))
	}
	if len(v95) != 4+2+len(fixtureSpw) {
		t.Errorf("gms_v95 body = %d bytes, want %d (Encode4 + EncodeStr)", len(v95), 4+2+len(fixtureSpw))
	}
	if !bytes.Equal(v83, v84) {
		t.Errorf("gms_v84 must take the gms_v83 path: %#v vs %#v", v84, v83)
	}
	if !bytes.Equal(v83, v92) {
		t.Errorf("gms_v92 must take the gms_v83 path: %#v vs %#v", v92, v83)
	}
}
