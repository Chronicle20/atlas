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
// MAPLELIFE_RESULT (clientbound) — CUICharacterSaleDlg::OnCheckDuplicatedIDResult.
//
// FOUR in-scope cells: gms_v83, gms_v87, gms_v92, gms_v95. gms_v84 is
// VERSION-ABSENT — no CUICharacterSaleDlg code path exists on that binary
// (derivation.md §4.3, independently re-confirmed) — no marker, no fixture.
//
// The `ida=` address on each marker is that version's RECEIVER, decompiled
// this pass: gms_v83 @0x7d768a, gms_v87 @0x82e12c, gms_v92 @0x756370,
// gms_v95 @0x777e40 — all four byte-for-byte identical in structure:
// DecodeStr(sName) then a SIGNED Decode1(nResult), three-way branch
// (derivation.md §4.7).
// ---------------------------------------------------------------------------

// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v83 ida=0x7d768a
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v87 ida=0x82e12c
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v92 ida=0x756370
// packet-audit:verify packet=maplelife/clientbound/MapleLifeResult version=gms_v95 ida=0x777e40
func TestMapleLifeResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMapleLifeResult("Chronicle", 1)
			output := MapleLifeResult{}
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

func TestMapleLifeResultOperation(t *testing.T) {
	m := MapleLifeResult{}
	if m.Operation() != MapleLifeResultWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), MapleLifeResultWriter)
	}
	if MapleLifeResultWriter != "MapleLifeResult" {
		t.Errorf("writer name = %q", MapleLifeResultWriter)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteAsciiString: uint16 LE length
//     prefix, then the raw bytes.
//   - WriteInt8: a single raw byte (the int8 value reinterpreted as uint8).
//
// Field order: derivation.md §4 (DecodeStr sName; Decode1 nResult SIGNED).
// The body is IDENTICAL on every in-scope version, so the fixtures are
// deliberately the same bytes for all four — that sameness IS the derived
// claim.
//
//	sName  "Chronicle" (9 chars) -> 09 00 43 68 72 6F 6E 69 63 6C 65
// ---------------------------------------------------------------------------

var mlrWireName = []byte{
	0x09, 0x00, // uint16 LE length = 9
	0x43, 0x68, 0x72, 0x6F, 0x6E, 0x69, 0x63, 0x6C, 0x65, // "Chronicle"
}

type mlrFixtureVersion struct {
	name  string
	major uint16
	minor uint16
}

// The four in-scope cells this row claims.
var mlrFixtureVersions = []mlrFixtureVersion{
	{"gms_v83", 83, 1},
	{"gms_v87", 87, 1},
	{"gms_v92", 92, 1},
	{"gms_v95", 95, 1},
}

func decodeMLRBytes(t *testing.T, name string, major uint16, minor uint16, body []byte) MapleLifeResult {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out MapleLifeResult
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

// TestMapleLifeResultByteFixture pins one case per §4 arm — taken (>0),
// available (==0), unknown error (<0) — for every in-scope version.
func TestMapleLifeResultByteFixture(t *testing.T) {
	cases := []struct {
		arm    string
		result int8
	}{
		{"taken", 1},
		{"available", 0},
		{"unknown_error", -1},
	}
	for _, v := range mlrFixtureVersions {
		for _, c := range cases {
			t.Run(v.name+"/"+c.arm, func(t *testing.T) {
				want := []byte{}
				want = append(want, mlrWireName...)
				want = append(want, byte(c.result))

				ctx := pt.CreateContext("GMS", v.major, v.minor)
				in := NewMapleLifeResult("Chronicle", c.result)
				got := pt.Encode(t, ctx, in.Encode, nil)
				if !bytes.Equal(got, want) {
					t.Fatalf("%s/%s encode:\n got %#v\nwant %#v", v.name, c.arm, got, want)
				}
				if len(got) != 12 {
					t.Errorf("%s/%s body = %d bytes, want 12 (2+9 DecodeStr + 1 Decode1)", v.name, c.arm, len(got))
				}

				out := decodeMLRBytes(t, v.name, v.major, v.minor, want)
				if out.Name() != "Chronicle" {
					t.Errorf("%s/%s name: got %q, want %q", v.name, c.arm, out.Name(), "Chronicle")
				}
				if out.Result() != c.result {
					t.Errorf("%s/%s result: got %d, want %d", v.name, c.arm, out.Result(), c.result)
				}
			})
		}
	}
}

// TestMapleLifeResultNoVersionDivergence pins the "identical on every
// in-scope version" claim explicitly (derivation.md §4.7: "No field, width,
// order, or branch-arm divergence across any present version"), so a future
// version gate added to this codec cannot pass silently.
func TestMapleLifeResultNoVersionDivergence(t *testing.T) {
	m := NewMapleLifeResult("Chronicle", 1)
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range mlrFixtureVersions {
		got := pt.Encode(t, pt.CreateContext("GMS", v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// ---------------------------------------------------------------------------
// Config-resolved result codes (DOM-25).
// ---------------------------------------------------------------------------

func mlrOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			MapleLifeResultAvailable:    float64(0),
			MapleLifeResultTaken:        float64(1),
			MapleLifeResultUnknownError: float64(0xFF),
		},
	}
}

func TestMapleLifeResultCodesAreConfigResolved(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want int8
	}{
		{MapleLifeResultAvailable, 0},
		{MapleLifeResultTaken, 1},
		{MapleLifeResultUnknownError, -1},
	} {
		t.Run(tc.key, func(t *testing.T) {
			body := MapleLifeResultBody("Chronicle", tc.key)
			got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, mlrOptions())
			wantLen := len("Chronicle") + 2 + 1
			if len(got) != wantLen {
				t.Fatalf("body = %d bytes, want %d", len(got), wantLen)
			}
			gotResult := int8(got[len(got)-1])
			if gotResult != tc.want {
				t.Errorf("resolved code = %d, want %d", gotResult, tc.want)
			}
		})
	}

	// An empty options map must NOT silently produce a valid-looking arm —
	// it resolves to ResolveCode's loud 99 sentinel.
	body := MapleLifeResultBody("Chronicle", MapleLifeResultTaken)
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, map[string]interface{}{})
	gotResult := int8(got[len(got)-1])
	if gotResult != 99 {
		t.Errorf("empty-options resolved code = %d, want 99 (ResolveCode sentinel)", gotResult)
	}

	// Re-resolving the SAME key against a template that configures a
	// different byte must follow the config, not a Go constant.
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			MapleLifeResultTaken: float64(7),
		},
	}
	body = MapleLifeResultBody("Chronicle", MapleLifeResultTaken)
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, remapped)
	gotResult = int8(got[len(got)-1])
	if gotResult != 7 {
		t.Errorf("remapped code = %d, want 7 — the byte is not config-resolved", gotResult)
	}
}

// TestMapleLifeResultReasonMapping pins the character.NameReason* -> wire-arm
// mapping. libs/atlas-packet must not import an atlas-channel package, so the
// four reasons are asserted as the literal strings
// services/atlas-channel/.../character/name_validity_requests.go declares:
// NameReasonLength="length", NameReasonRegex="regex",
// NameReasonDuplicate="duplicate", NameReasonReserved="reserved".
func TestMapleLifeResultReasonMapping(t *testing.T) {
	const (
		nameReasonLength    = "length"
		nameReasonRegex     = "regex"
		nameReasonDuplicate = "duplicate"
		nameReasonReserved  = "reserved"
	)
	required := []string{nameReasonLength, nameReasonRegex, nameReasonDuplicate, nameReasonReserved}

	if len(mapleLifeResultReasonArms) != len(required) {
		t.Fatalf("reason table has %d entries, want %d — the taxonomy is closed", len(mapleLifeResultReasonArms), len(required))
	}

	wantArm := map[string]string{
		nameReasonLength:    MapleLifeResultUnknownError,
		nameReasonRegex:     MapleLifeResultUnknownError,
		nameReasonDuplicate: MapleLifeResultTaken,
		nameReasonReserved:  MapleLifeResultUnknownError,
	}
	wantWire := map[string]int8{
		nameReasonLength:    -1,
		nameReasonRegex:     -1,
		nameReasonDuplicate: 1,
		nameReasonReserved:  -1,
	}

	for _, reason := range required {
		key, ok := mapleLifeResultReasonArms[reason]
		if !ok {
			t.Fatalf("reason %q missing from the mapping table", reason)
		}
		if key != wantArm[reason] {
			t.Errorf("reason %q maps to %q, want %q", reason, key, wantArm[reason])
		}

		body := MapleLifeResultRejectedBody("Chronicle", reason)
		got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, mlrOptions())
		gotResult := int8(got[len(got)-1])
		if gotResult != wantWire[reason] {
			t.Errorf("%s resolved code = %d, want %d", reason, gotResult, wantWire[reason])
		}
	}

	// duplicate is the ONLY reason with a semantically exact arm.
	if wantArm[nameReasonDuplicate] == MapleLifeResultUnknownError {
		t.Fatal("duplicate must not collapse to UNKNOWN_ERROR — this op's exact-match arm")
	}

	// A reason outside the closed taxonomy must still produce the safe
	// unknown-error arm rather than a guessed byte.
	body := MapleLifeResultRejectedBody("Chronicle", "not_a_reason")
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, mlrOptions())
	gotResult := int8(got[len(got)-1])
	if gotResult != -1 {
		t.Errorf("unknown reason resolved code = %d, want -1 (UNKNOWN_ERROR)", gotResult)
	}
}
