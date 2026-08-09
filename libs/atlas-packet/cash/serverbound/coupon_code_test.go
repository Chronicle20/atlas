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

// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v48 ida=0x44d2e7
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v61 ida=0x45a6b5
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v72 ida=0x4698d8
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v79 ida=0x46aa3e
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v83 ida=0x4710e8
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v84 ida=0x473bde
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v87 ida=0x47b9d4
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v92 ida=0x484430
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v95 ida=0x487ee0
// packet-audit:verify packet=cash/serverbound/CouponCode version=jms_v185 ida=0x482450
func TestCouponCodeRoundTripSelfRedeem(t *testing.T) {
	// The plain-redeem path the client actually takes: targetCharacter empty, so
	// the conditional third string is never emitted on any version. jms_v185
	// still carries its unconditional nType byte between field 2 and the (here
	// absent) third string.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CouponCode{targetCharacter: "", code: "MAPLE2026"}
			output := CouponCode{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != "" {
				t.Errorf("targetCharacter: got %q, want empty", output.TargetCharacter())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %q, want %q", output.Code(), input.Code())
			}
			if output.Extra() != "" {
				t.Errorf("extra: got %q, want empty", output.Extra())
			}
			if output.Type() != 0 {
				t.Errorf("type: got %d, want 0", output.Type())
			}
		})
	}
}

func TestCouponCodeRoundTripTargetedRedeem(t *testing.T) {
	// targetCharacter non-empty: gms_v48..v87 and jms_v185 add the guarded third
	// string; gms_v92/v95 have no third string at all, so extra decodes back
	// empty there. jms_v185 additionally round-trips nType.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CouponCode{targetCharacter: "Sidekick", code: "MAPLE2026", nType: 7, extra: "EXTRA"}
			output := CouponCode{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != input.TargetCharacter() {
				t.Errorf("targetCharacter: got %q, want %q", output.TargetCharacter(), input.TargetCharacter())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %q, want %q", output.Code(), input.Code())
			}
			wantExtra := input.Extra()
			if !hasTrailingCouponString(ctx) {
				wantExtra = ""
			}
			if output.Extra() != wantExtra {
				t.Errorf("extra: got %q, want %q for %s", output.Extra(), wantExtra, v.Name)
			}
			wantType := input.Type()
			if !hasCouponType(ctx) {
				wantType = 0
			}
			if output.Type() != wantType {
				t.Errorf("type: got %d, want %d for %s", output.Type(), wantType, v.Name)
			}
		})
	}
}

// TestCouponCodeJmsCarriesTypeByte pins the jms_v185 divergence derived at
// COutPacket::Encode1 @ 0x4825e5: the nType byte sits between the coupon code
// and the guarded third string, and is on the wire unconditionally. Collapsing
// jms onto the GMS shape would desync the decoder by exactly one byte.
func TestCouponCodeJmsCarriesTypeByte(t *testing.T) {
	gms := pt.CreateContext("GMS", 83, 1)
	jms := pt.CreateContext("JMS", 185, 1)

	m := CouponCode{targetCharacter: "", code: "AB", nType: 9}
	gmsBytes := pt.Encode(t, gms, m.Encode, nil)
	jmsBytes := pt.Encode(t, jms, m.Encode, nil)

	if len(jmsBytes) != len(gmsBytes)+1 {
		t.Fatalf("jms body length %d, want gms length %d + 1", len(jmsBytes), len(gmsBytes))
	}
	if jmsBytes[len(jmsBytes)-1] != 9 {
		t.Errorf("jms trailing byte = %d, want the nType value 9", jmsBytes[len(jmsBytes)-1])
	}
}

// TestCouponCodeStringDoesNotLeakCode - coupon codes are secrets. String() must
// report the code's LENGTH and never its value.
func TestCouponCodeStringDoesNotLeakCode(t *testing.T) {
	m := CouponCode{targetCharacter: "Sidekick", code: "SUPERSECRET1"}
	s := m.String()
	if strings.Contains(s, "SUPERSECRET1") {
		t.Errorf("String() leaked the coupon code: %s", s)
	}
	if !strings.Contains(s, "12") {
		t.Errorf("String() = %q, want it to report the code length (12)", s)
	}
}

func TestCouponCodeOperation(t *testing.T) {
	if (CouponCode{}).Operation() != CashShopCouponCodeHandle {
		t.Errorf("Operation() = %q, want %q", (CouponCode{}).Operation(), CashShopCouponCodeHandle)
	}
	if CashShopCouponCodeHandle != "CashShopCouponCodeHandle" {
		t.Errorf("CashShopCouponCodeHandle = %q, want CashShopCouponCodeHandle", CashShopCouponCodeHandle)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// The two round-trip tests above are SYMMETRIC: encode and decode move
// together, so a change to the shared string framing (e.g. the uint16
// length prefix narrowing to a byte) keeps them green while every real coupon
// submission desyncs on the wire. The fixtures below close that hole by
// pinning the exact bytes.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteAsciiString: WriteShort(len)
//     then the Shift-JIS bytes; WriteShort is binary.LittleEndian (writer.go:43).
//   - libs/atlas-socket/request/reader.go ReadAsciiString: ReadInt16 then
//     ReadString(n), decoded Shift-JIS.
//
// Every literal below is ASCII-only, where Shift-JIS is byte-identical to
// ASCII, so each string contributes `lo hi` + its ASCII bytes.
//
// Field order per version comes from
// docs/tasks/task-206-cash-shop-coupon-codes/derivation.md; the same three
// shapes cover all ten versions.
// ---------------------------------------------------------------------------

// couponFixtureVersion is one of the ten versions this task claims. The byte
// fixtures are keyed on these explicitly rather than on pt.Variants: the claim
// is about the ten versions with a registry entry, an IDA-pinned evidence
// record and a matrix cell — not about every variant that happens to be in the
// shared table.
type couponFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
	// thirdString: the version's send path contains the guarded third
	// EncodeStr (gms_v48..v87 and jms_v185; absent on gms_v92/v95).
	thirdString bool
	// typeByte: the version emits the unconditional 1-byte nType (jms only).
	typeByte bool
}

var couponFixtureVersions = []couponFixtureVersion{
	{"gms_v48", "GMS", 48, 1, true, false},
	{"gms_v61", "GMS", 61, 1, true, false},
	{"gms_v72", "GMS", 72, 1, true, false},
	{"gms_v79", "GMS", 79, 1, true, false},
	{"gms_v83", "GMS", 83, 1, true, false},
	{"gms_v84", "GMS", 84, 1, true, false},
	{"gms_v87", "GMS", 87, 1, true, false},
	{"gms_v92", "GMS", 92, 1, false, false},
	{"gms_v95", "GMS", 95, 1, false, false},
	{"jms_v185", "JMS", 185, 1, true, true},
}

// Fixture field values, and their hand-derived wire bytes.
//
//	targetCharacter "Bob"  -> 03 00 42 6F 62      ('B'=0x42 'o'=0x6F 'b'=0x62)
//	targetCharacter ""     -> 00 00
//	code            "AB12" -> 04 00 41 42 31 32   ('A'=0x41 'B'=0x42 '1'=0x31 '2'=0x32)
//	extra           "XY"   -> 02 00 58 59         ('X'=0x58 'Y'=0x59)
//	nType           7      -> 07
const (
	fixtureTarget = "Bob"
	fixtureCode   = "AB12"
	fixtureExtra  = "XY"
	fixtureType   = byte(7)
)

var (
	wireEmptyTarget = []byte{0x00, 0x00}
	wireTarget      = []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	wireCode        = []byte{0x04, 0x00, 0x41, 0x42, 0x31, 0x32}
	wireExtra       = []byte{0x02, 0x00, 0x58, 0x59}
	wireType        = []byte{0x07}
)

func concatBytes(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// decodeCouponBytes feeds a raw body to Decode, asserting the decoder consumes
// it exactly — a decoder that stops one byte early (the jms nType hazard) or
// runs off the end fails here rather than silently.
func decodeCouponBytes(t *testing.T, v couponFixtureVersion, body []byte) CouponCode {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(v.region, v.major, v.minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CouponCode
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", v.name, reader.Available(), len(body))
	}
	return out
}

// TestCouponCodeByteFixtureSelfRedeem pins the plain-redeem wire body — the
// path a real client takes — for all ten versions. targetCharacter is empty,
// so the guarded third string is emitted by NO version; the gms_v48..v87 and
// gms_v92/v95 shapes are therefore byte-identical here, and jms differs by
// exactly its nType byte.
func TestCouponCodeByteFixtureSelfRedeem(t *testing.T) {
	for _, v := range couponFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := concatBytes(wireEmptyTarget, wireCode)
			if v.typeByte {
				want = concatBytes(want, wireType)
			}

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := CouponCode{targetCharacter: "", code: fixtureCode, nType: fixtureType, extra: fixtureExtra}
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}

			out := decodeCouponBytes(t, v, want)
			if out.TargetCharacter() != "" {
				t.Errorf("%s targetCharacter: got %q, want empty", v.name, out.TargetCharacter())
			}
			if out.Code() != fixtureCode {
				t.Errorf("%s code: got %q, want %q", v.name, out.Code(), fixtureCode)
			}
			if out.Extra() != "" {
				t.Errorf("%s extra: got %q, want empty", v.name, out.Extra())
			}
			wantType := byte(0)
			if v.typeByte {
				wantType = fixtureType
			}
			if out.Type() != wantType {
				t.Errorf("%s type: got %d, want %d", v.name, out.Type(), wantType)
			}
		})
	}
}

// TestCouponCodeByteFixtureTargetedRedeem pins the populated-field-1 body,
// where the three shapes actually diverge: gms_v48..v87 append the guarded
// third string, gms_v92/v95 have no third string at all, and jms carries both
// its nType byte AND the guarded third string — in that order. This is the
// case a framing change would hide, because the guard itself is what decides
// how many strings are on the wire.
func TestCouponCodeByteFixtureTargetedRedeem(t *testing.T) {
	for _, v := range couponFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := concatBytes(wireTarget, wireCode)
			if v.typeByte {
				want = concatBytes(want, wireType)
			}
			if v.thirdString {
				want = concatBytes(want, wireExtra)
			}

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := CouponCode{targetCharacter: fixtureTarget, code: fixtureCode, nType: fixtureType, extra: fixtureExtra}
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}

			out := decodeCouponBytes(t, v, want)
			if out.TargetCharacter() != fixtureTarget {
				t.Errorf("%s targetCharacter: got %q, want %q", v.name, out.TargetCharacter(), fixtureTarget)
			}
			if out.Code() != fixtureCode {
				t.Errorf("%s code: got %q, want %q", v.name, out.Code(), fixtureCode)
			}
			wantExtra := ""
			if v.thirdString {
				wantExtra = fixtureExtra
			}
			if out.Extra() != wantExtra {
				t.Errorf("%s extra: got %q, want %q", v.name, out.Extra(), wantExtra)
			}
			wantType := byte(0)
			if v.typeByte {
				wantType = fixtureType
			}
			if out.Type() != wantType {
				t.Errorf("%s type: got %d, want %d", v.name, out.Type(), wantType)
			}
		})
	}
}

// TestCouponCodeByteFixtureShapesDiffer guards the fixtures themselves: if a
// future edit collapsed the three shapes onto one body, the two tests above
// would still pass while asserting nothing. This pins that the targeted-redeem
// bodies are genuinely distinct across the three shapes, and pins the exact
// jms byte offset of nType.
func TestCouponCodeByteFixtureShapesDiffer(t *testing.T) {
	body := func(name string) []byte {
		for _, v := range couponFixtureVersions {
			if v.name != name {
				continue
			}
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := CouponCode{targetCharacter: fixtureTarget, code: fixtureCode, nType: fixtureType, extra: fixtureExtra}
			return pt.Encode(t, ctx, in.Encode, nil)
		}
		t.Fatalf("unknown fixture version %s", name)
		return nil
	}

	three, two, jms := body("gms_v83"), body("gms_v92"), body("jms_v185")
	if bytes.Equal(three, two) {
		t.Errorf("gms_v83 and gms_v92 bodies are identical (%#v); v92 must not emit the third string", three)
	}
	if bytes.Equal(three, jms) {
		t.Errorf("gms_v83 and jms_v185 bodies are identical (%#v); jms must carry the nType byte", three)
	}
	// nType sits immediately after the coupon code: len(target)+len(code) bytes in.
	off := len(wireTarget) + len(wireCode)
	if jms[off] != fixtureType {
		t.Errorf("jms nType at offset %d = 0x%02X, want 0x%02X (body %#v)", off, jms[off], fixtureType, jms)
	}
	if !bytes.Equal(jms, concatBytes(three[:off], wireType, three[off:])) {
		t.Errorf("jms body is not the gms_v83 body with nType spliced at offset %d:\n jms %#v\n v83 %#v", off, jms, three)
	}
}
