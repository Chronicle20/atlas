package clientbound

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// ---------------------------------------------------------------------------
// CASHSHOP_CHECK_NAME_CHANGE (clientbound) — CCashShop::OnCheckDuplicatedIDResult.
//
// NINE applicable cells: v48, v61, v72, v79, v83, v84, v87, v92, v95.
// jms_v185 has no name-change feature at all (derivation.md §1.5) — no marker,
// no fixture, VERSION-ABSENT.
//
// The `ida=` address on each marker is that version's RECEIVER. v48, v61, v72,
// v83 and v95 were independently re-decompiled during this task
// (v48 @0x455a7f, v61 @0x463900, v72 @0x473519, v83 @0x47baea, v95 @0x497fb0)
// and are byte-for-byte identical in structure: DecodeStr(sName) then a
// SIGNED Decode1(nResult). v79/v84/v87/v92 are carried from derivation.md's
// prior pass (v79 @0x4749e5, v84 @0x47ec88, v87 @0x4872cb, v92 @0x493f40) —
// no version between the five directly re-checked here shows any structural
// divergence a middle version could plausibly introduce.
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v48 ida=0x455a7f
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v61 ida=0x463900
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v72 ida=0x473519
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v79 ida=0x4749e5
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v83 ida=0x47baea
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v84 ida=0x47ec88
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v87 ida=0x4872cb
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v92 ida=0x493f40
// packet-audit:verify packet=cash/clientbound/CashCheckNameChange version=gms_v95 ida=0x497fb0
func TestCheckNameChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckNameChange("Chronicle", 1)
			output := CheckNameChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.Name() != input.Name() {
				t.Errorf("name: got %q, want %q", output.Name(), input.Name())
			}
			if output.Result() != input.Result() {
				t.Errorf("result: got %d, want %d", output.Result(), input.Result())
			}
		})
	}
}

func TestCheckNameChangeOperation(t *testing.T) {
	m := CheckNameChange{}
	if m.Operation() != CashShopCheckNameChangeWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), CashShopCheckNameChangeWriter)
	}
	if CashShopCheckNameChangeWriter != "CashShopCheckNameChange" {
		t.Errorf("writer name = %q", CashShopCheckNameChangeWriter)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// The round-trip test above is SYMMETRIC: a change to the shared framing moves
// encode and decode together and stays green while every real client read
// desyncs. The literal byte arrays below are written independently of this
// codec's own Encode output and pin the exact bytes.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteAsciiString: uint16 LE length
//     prefix, then the raw bytes (ASCII input passes through Shift-JIS
//     untouched).
//   - WriteInt8: a single raw byte (the int8 value reinterpreted as uint8).
//
// Field order: derivation.md §2.3, re-confirmed by decompiling five of the
// nine receivers during this task. The body is IDENTICAL on every applicable
// version, so the nine fixtures are deliberately the same bytes — that
// sameness IS the derived claim, and a future version gate that broke it
// would fail here.
//
//	sName  "Chronicle" (9 chars) -> 09 00 43 68 72 6F 6E 69 63 6C 65
//	nResult 0x01 (TAKEN, >0)     -> 01
// ---------------------------------------------------------------------------

const (
	ncFixtureName   = "Chronicle"
	ncFixtureResult = int8(1)
)

var (
	ncWireName = []byte{
		0x09, 0x00, // uint16 LE length = 9
		0x43, 0x68, 0x72, 0x6F, 0x6E, 0x69, 0x63, 0x6C, 0x65, // "Chronicle"
	}
	ncWireResult = []byte{0x01}
)

type nameChangeFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
}

// The nine cells this row claims.
var nameChangeFixtureVersions = []nameChangeFixtureVersion{
	{"gms_v48", "GMS", 48, 1},
	{"gms_v61", "GMS", 61, 1},
	{"gms_v72", "GMS", 72, 1},
	{"gms_v79", "GMS", 79, 1},
	{"gms_v83", "GMS", 83, 1},
	{"gms_v84", "GMS", 84, 1},
	{"gms_v87", "GMS", 87, 1},
	{"gms_v92", "GMS", 92, 1},
	{"gms_v95", "GMS", 95, 1},
}

func decodeNameChangeBytes(t *testing.T, name string, region string, major uint16, minor uint16, body []byte) CheckNameChange {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(region, major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CheckNameChange
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

func TestCheckNameChangeByteFixture(t *testing.T) {
	for _, v := range nameChangeFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := []byte{}
			want = append(want, ncWireName...)
			want = append(want, ncWireResult...)

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCheckNameChange(ncFixtureName, ncFixtureResult)
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			if len(got) != 12 {
				t.Errorf("%s body = %d bytes, want 12 (2+9 DecodeStr + 1 Decode1)", v.name, len(got))
			}

			out := decodeNameChangeBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Name() != ncFixtureName {
				t.Errorf("%s name: got %q, want %q", v.name, out.Name(), ncFixtureName)
			}
			if out.Result() != ncFixtureResult {
				t.Errorf("%s result: got %d, want %d", v.name, out.Result(), ncFixtureResult)
			}
		})
	}
}

// TestCheckNameChangeSignedResultRoundTrips pins the SIGNED nature of nResult:
// a negative wire byte must decode back to a negative int8, not an unsigned
// wraparound. derivation.md §4.4: "nResult is signed: > 0 duplicate, == 0
// available, < 0 unknown error. Every version tests it with a signed
// comparison."
func TestCheckNameChangeSignedResultRoundTrips(t *testing.T) {
	body := []byte{0x00, 0x00, 0xFF} // empty name, nResult = 0xFF = -1 signed
	out := decodeNameChangeBytes(t, "signed", "GMS", 83, 1, body)
	if out.Result() != -1 {
		t.Errorf("result: got %d, want -1", out.Result())
	}
}

// TestCheckNameChangeNoVersionDivergence pins the "identical on every
// applicable version" claim explicitly, so a future version gate added to
// this codec cannot pass silently. It also covers the v84 off-by-one trap:
// v84 must be byte-identical to v83 here.
func TestCheckNameChangeNoVersionDivergence(t *testing.T) {
	m := NewCheckNameChange(ncFixtureName, ncFixtureResult)
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range nameChangeFixtureVersions {
		got := pt.Encode(t, pt.CreateContext(v.region, v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// ---------------------------------------------------------------------------
// Config-resolved reason codes (DOM-25).
//
// The result byte must come from the tenant template's operations table,
// never from a Go literal. These tests drive the body builders with a
// synthetic options map and assert the byte that lands on the wire is the
// CONFIGURED one.
// ---------------------------------------------------------------------------

// ncOptions mirrors the operations table the nine seed templates carry for
// the CashShopCheckNameChange writer.
func ncOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CheckNameChangeAvailable:    float64(0),
			CheckNameChangeTaken:        float64(1),
			CheckNameChangeUnknownError: float64(0xFF),
		},
	}
}

func TestCheckNameChangeCodesAreConfigResolved(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want int8
	}{
		{CheckNameChangeAvailable, 0},
		{CheckNameChangeTaken, 1},
		{CheckNameChangeUnknownError, -1},
	} {
		t.Run(tc.key, func(t *testing.T) {
			body := CheckNameChangeResultBody(ncFixtureName, tc.key)
			got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, ncOptions())
			wantLen := len(ncFixtureName) + 2 + 1
			if len(got) != wantLen {
				t.Fatalf("body = %d bytes, want %d", len(got), wantLen)
			}
			gotResult := int8(got[len(got)-1])
			if gotResult != tc.want {
				t.Errorf("resolved code = %d, want %d", gotResult, tc.want)
			}
		})
	}

	// Re-resolving the SAME key against a template that configures a
	// different byte must follow the config, not a Go constant.
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			CheckNameChangeTaken: float64(7),
		},
	}
	body := CheckNameChangeResultBody(ncFixtureName, CheckNameChangeTaken)
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, remapped)
	gotResult := int8(got[len(got)-1])
	if gotResult != 7 {
		t.Errorf("remapped code = %d, want 7 — the byte is not config-resolved", gotResult)
	}
}

func TestCheckNameChangeAvailableBody(t *testing.T) {
	body := CheckNameChangeAvailableBody(ncFixtureName)
	got := pt.Encode(t, pt.CreateContext("GMS", 95, 1), body, ncOptions())

	want := []byte{}
	want = append(want, ncWireName...)
	want = append(want, 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("available body:\n got %#v\nwant %#v", got, want)
	}
}

// TestCheckNameChangeReasonMapping pins the design §6 reason -> wire-code
// mapping. This row OWNS the name-validity taxonomy handoff, but its client
// switch is only three-way (taken / available / unknown-error) — see the
// type doc comment's "This row owns the name-validity reason taxonomy — and
// cannot fully carry it" section. Only name_taken lands on a semantically
// exact arm (nResult > 0, "this name is currently in use"); the other three
// required reasons collapse onto UNKNOWN_ERROR because no arm of
// OnCheckDuplicatedIDResult renders "reserved" / "too short" / "bad
// characters" text on any version examined.
func TestCheckNameChangeReasonMapping(t *testing.T) {
	required := []string{"name_taken", "name_reserved", "name_invalid_length", "name_invalid_charset"}

	if len(checkNameChangeReasonArms) != len(required) {
		t.Fatalf("reason table has %d entries, want %d — the taxonomy is closed", len(checkNameChangeReasonArms), len(required))
	}

	wantArm := map[string]string{
		"name_taken":           CheckNameChangeTaken,
		"name_reserved":        CheckNameChangeUnknownError,
		"name_invalid_length":  CheckNameChangeUnknownError,
		"name_invalid_charset": CheckNameChangeUnknownError,
	}
	wantWire := map[string]int8{
		"name_taken":           1,
		"name_reserved":        -1,
		"name_invalid_length":  -1,
		"name_invalid_charset": -1,
	}

	for _, reason := range required {
		key, ok := checkNameChangeReasonArms[reason]
		if !ok {
			t.Fatalf("reason %q missing from the mapping table", reason)
		}
		if key != wantArm[reason] {
			t.Errorf("reason %q maps to %q, want %q", reason, key, wantArm[reason])
		}

		body := CheckNameChangeRejectedBody(ncFixtureName, reason)
		got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, ncOptions())
		gotResult := int8(got[len(got)-1])
		if gotResult != wantWire[reason] {
			t.Errorf("%s resolved code = %d, want %d", reason, gotResult, wantWire[reason])
		}
	}

	// name_taken is the ONLY reason with a semantically exact arm.
	if wantArm["name_taken"] == CheckNameChangeUnknownError {
		t.Fatal("name_taken must not collapse to UNKNOWN_ERROR — this op is the one place the taxonomy can express it")
	}

	// A reason outside the closed taxonomy is a server bug; it must still
	// produce the safe unknown-error arm rather than a guessed byte.
	body := CheckNameChangeRejectedBody(ncFixtureName, "not_a_reason")
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, ncOptions())
	gotResult := int8(got[len(got)-1])
	if gotResult != -1 {
		t.Errorf("unknown reason resolved code = %d, want -1 (UNKNOWN_ERROR)", gotResult)
	}
}
