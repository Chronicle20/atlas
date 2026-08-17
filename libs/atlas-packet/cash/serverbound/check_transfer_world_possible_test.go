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
// WORLD_TRANSFER (serverbound) — CCashShop::SendCheckTransferWorldPossiblePacket.
//
// TEN applicable cells: the nine GMS versions plus jms_v185. Unlike its
// NAME_TRANSFER sibling, this op DOES exist on jms_v185 — that client has no
// name-change feature at all, and its serverbound 0x009 (previously misfiled as
// NAME_TRANSFER on a transposed IDB symbol) is this packet
// (derivation.md §1.5).
//
// The `ida=` address on each marker is that version's send site, read out of
// the version's own IDB (derivation.md §1.2) and re-confirmed by decompiling
// each one during this task:
//
//	gms_v48   COutPacket::COutPacket((COutPacket *)v5, 20)  @0x44fbac  -> 0x014
//	gms_v61…gms_v95  COutPacket(…, 18)                                 -> 0x012
//	jms_v185  COutPacket::COutPacket(v6, 9)                 @0x484fd8  -> 0x009
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v48 ida=0x44fb95
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v61 ida=0x45cefd
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v72 ida=0x46be19
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v79 ida=0x46cf90
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v83 ida=0x47359c
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v84 ida=0x4760e6
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v87 ida=0x47df1a
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v92 ida=0x47f8c0
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=gms_v95 ida=0x4884c0
// packet-audit:verify packet=cash/serverbound/CashCheckTransferWorldPossible version=jms_v185 ida=0x484fbf
func TestCheckTransferWorldPossibleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckTransferWorldPossible(123456, 19900102, "19900102")
			output := CheckTransferWorldPossible{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if transferBodyHasCharacterId(ctx) {
				if output.CharacterId() != input.CharacterId() {
					t.Errorf("characterId: got %d, want %d", output.CharacterId(), input.CharacterId())
				}
			} else if output.CharacterId() != 0 {
				t.Errorf("characterId: got %d, want 0 (this client sends no character id)", output.CharacterId())
			}

			if transferCredentialIsString(ctx) {
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

// TestCheckTransferWorldPossibleStringRedactsCredential — the credential field
// is the account second password / birthday code, and every serverbound handler
// in atlas-channel logs p.String() at debug level. Neither form of the
// credential may appear in that line, in whole or in part.
func TestCheckTransferWorldPossibleStringRedactsCredential(t *testing.T) {
	m := NewCheckTransferWorldPossible(777, 19771231, "19771231")
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

func TestCheckTransferWorldPossibleOperation(t *testing.T) {
	if (CheckTransferWorldPossible{}).Operation() != CashShopCheckTransferWorldPossibleHandle {
		t.Errorf("Operation() = %q, want %q", (CheckTransferWorldPossible{}).Operation(), CashShopCheckTransferWorldPossibleHandle)
	}
	if CashShopCheckTransferWorldPossibleHandle != "CashShopCheckTransferWorldPossibleHandle" {
		t.Errorf("CashShopCheckTransferWorldPossibleHandle = %q", CashShopCheckTransferWorldPossibleHandle)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// The round-trip test above is SYMMETRIC: a change to the shared framing moves
// encode and decode together and stays green while every real client request
// desyncs. The literal byte arrays below are written independently of this
// codec's own Encode output and pin the exact bytes per version.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteInt: binary.LittleEndian uint32.
//   - WriteAsciiString: WriteShort(len) little-endian, then the Shift-JIS bytes
//     (ASCII-identical for the digits used here).
//   - libs/atlas-socket/request/reader.go mirrors both.
//
// Field order per version: derivation.md §2.2, re-confirmed by decompiling each
// send site during this task — two Encode4 on gms_v48…gms_v92; Encode4 +
// EncodeStr on gms_v95; a single EncodeStr on jms_v185.
// ---------------------------------------------------------------------------

type worldTransferFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
	// hasCharacterId: the body opens with Encode4 dwCharacterID. True on every
	// GMS version; false on jms_v185, whose body is the credential alone.
	hasCharacterId bool
	// stringCredential: the version encodes the credential with EncodeStr
	// (gms_v95 and jms_v185); every earlier GMS version uses Encode4.
	stringCredential bool
}

// The ten cells this task claims.
var worldTransferFixtureVersions = []worldTransferFixtureVersion{
	{"gms_v48", "GMS", 48, 1, true, false},
	{"gms_v61", "GMS", 61, 1, true, false},
	{"gms_v72", "GMS", 72, 1, true, false},
	{"gms_v79", "GMS", 79, 1, true, false},
	{"gms_v83", "GMS", 83, 1, true, false},
	{"gms_v84", "GMS", 84, 1, true, false},
	{"gms_v87", "GMS", 87, 1, true, false},
	{"gms_v92", "GMS", 92, 1, true, false},
	{"gms_v95", "GMS", 95, 1, true, true},
	{"jms_v185", "JMS", 185, 1, false, true},
}

// Fixture values and their hand-derived wire bytes.
//
//	characterId 0x0001E240 (123456)   -> 40 E2 01 00
//	birthDate   0x012FA6C6 (19900102) -> C6 A6 2F 01
//	spw         "19900102"            -> 08 00 31 39 39 30 30 31 30 32
const (
	wtFixtureCharacterId = uint32(123456)
	wtFixtureBirthDate   = uint32(19900102)
	wtFixtureSpw         = "19900102"
)

var (
	wtWireCharacterId = []byte{0x40, 0xE2, 0x01, 0x00}
	wtWireBirthDate   = []byte{0xC6, 0xA6, 0x2F, 0x01}
	wtWireSpw         = []byte{0x08, 0x00, 0x31, 0x39, 0x39, 0x30, 0x30, 0x31, 0x30, 0x32}
)

func decodeWorldTransferBytes(t *testing.T, v worldTransferFixtureVersion, body []byte) CheckTransferWorldPossible {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(v.region, v.major, v.minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CheckTransferWorldPossible
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", v.name, reader.Available(), len(body))
	}
	return out
}

func TestCheckTransferWorldPossibleByteFixture(t *testing.T) {
	for _, v := range worldTransferFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := []byte{}
			if v.hasCharacterId {
				want = append(want, wtWireCharacterId...)
			}
			if v.stringCredential {
				want = append(want, wtWireSpw...)
			} else {
				want = append(want, wtWireBirthDate...)
			}

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCheckTransferWorldPossible(wtFixtureCharacterId, wtFixtureBirthDate, wtFixtureSpw)
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}

			out := decodeWorldTransferBytes(t, v, want)
			if v.hasCharacterId {
				if out.CharacterId() != wtFixtureCharacterId {
					t.Errorf("%s characterId: got %d, want %d", v.name, out.CharacterId(), wtFixtureCharacterId)
				}
			} else if out.CharacterId() != 0 {
				t.Errorf("%s characterId: got %d, want 0", v.name, out.CharacterId())
			}
			if v.stringCredential {
				if out.Spw() != wtFixtureSpw {
					t.Errorf("%s spw: got %q, want %q", v.name, out.Spw(), wtFixtureSpw)
				}
			} else if out.BirthDate() != wtFixtureBirthDate {
				t.Errorf("%s birthDate: got %d, want %d", v.name, out.BirthDate(), wtFixtureBirthDate)
			}
		})
	}
}

// TestCheckTransferWorldPossibleV95ChangesWireType pins the GMS boundary
// explicitly: the credential is not merely widened at v95, it changes wire type
// from Encode4 to EncodeStr. A gate written as `> 83` (rather than
// MajorAtLeast(95)) would route v84 — and everything up to v92 — down the
// string path and desync every request.
func TestCheckTransferWorldPossibleV95ChangesWireType(t *testing.T) {
	m := NewCheckTransferWorldPossible(wtFixtureCharacterId, wtFixtureBirthDate, wtFixtureSpw)

	v92 := pt.Encode(t, pt.CreateContext("GMS", 92, 1), m.Encode, nil)
	v95 := pt.Encode(t, pt.CreateContext("GMS", 95, 1), m.Encode, nil)
	v84 := pt.Encode(t, pt.CreateContext("GMS", 84, 1), m.Encode, nil)
	v83 := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	v48 := pt.Encode(t, pt.CreateContext("GMS", 48, 1), m.Encode, nil)

	if len(v92) != 8 {
		t.Errorf("gms_v92 body = %d bytes, want 8 (Encode4 + Encode4)", len(v92))
	}
	if len(v95) != 4+2+len(wtFixtureSpw) {
		t.Errorf("gms_v95 body = %d bytes, want %d (Encode4 + EncodeStr)", len(v95), 4+2+len(wtFixtureSpw))
	}
	if !bytes.Equal(v83, v84) {
		t.Errorf("gms_v84 must take the gms_v83 path: %#v vs %#v", v84, v83)
	}
	if !bytes.Equal(v83, v92) {
		t.Errorf("gms_v92 must take the gms_v83 path: %#v vs %#v", v92, v83)
	}
	// v48 differs from v83 only in its OPCODE (0x014 vs 0x012), which lives in
	// the tenant template, not in the body. The body itself is identical.
	if !bytes.Equal(v48, v83) {
		t.Errorf("gms_v48 body must equal gms_v83: %#v vs %#v", v48, v83)
	}
}

// TestCheckTransferWorldPossibleJmsHasNoCharacterId pins the second, structural
// divergence: jms_v185's body is the credential ALONE. `retn 4` @0x485035
// proves the send takes a single stack argument, and there is no Encode4
// anywhere in CCashShop::SendCheckTransferWorldPossiblePacket @0x484fbf. A
// codec that reused the GMS shape here would read the credential's 2-byte
// length prefix as the low half of a character id and desync the rest of the
// stream.
func TestCheckTransferWorldPossibleJmsHasNoCharacterId(t *testing.T) {
	m := NewCheckTransferWorldPossible(wtFixtureCharacterId, wtFixtureBirthDate, wtFixtureSpw)

	jms := pt.Encode(t, pt.CreateContext("JMS", 185, 1), m.Encode, nil)
	if !bytes.Equal(jms, wtWireSpw) {
		t.Fatalf("jms_v185 body:\n got %#v\nwant %#v (EncodeStr sSPW only)", jms, wtWireSpw)
	}
	if len(jms) != 2+len(wtFixtureSpw) {
		t.Errorf("jms_v185 body = %d bytes, want %d", len(jms), 2+len(wtFixtureSpw))
	}

	gms95 := pt.Encode(t, pt.CreateContext("GMS", 95, 1), m.Encode, nil)
	if bytes.Equal(jms, gms95) {
		t.Error("jms_v185 must NOT share the gms_v95 shape — it carries no character id")
	}
	if len(gms95)-len(jms) != 4 {
		t.Errorf("gms_v95 must be exactly 4 bytes (the character id) longer than jms_v185: %d vs %d", len(gms95), len(jms))
	}
}
