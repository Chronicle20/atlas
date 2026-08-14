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
// CANCEL_NAME_CHANGE_RESULT (clientbound) — CWvsContext::OnCancelNameChangeResult.
//
// EIGHT applicable cells. Absent on gms_v48 (no receiver — the name-change
// feature and its cancel flow both arrive with the family system at v61+) and
// jms_v185 (no name-change feature at all, derivation.md §1.5).
//
// This is a NOTIFICATION: no version has a serverbound counterpart. Decode is
// carried anyway, per the standing rule, so the fixture test proves
// byte-exactness independent of Encode.
//
// The `ida=` address on each marker is that version's RECEIVER, read out of
// the version's own IDB. v61 and v83 were re-decompiled during Task 20; the
// remaining six were re-confirmed byte-for-byte identical via v92 and v95
// spot re-decompiles plus derivation.md §2.6's independent per-version pass:
//
//	gms_v61  0x84ace9   gms_v72  0x922399   gms_v79  0x9744ce
//	gms_v83  0xa2a677   gms_v84  0xa75e3a   gms_v87  0xac2313
//	gms_v92  0x9d64a0   gms_v95  0xa01b10
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v61 ida=0x84ace9
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v72 ida=0x922399
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v79 ida=0x9744ce
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v83 ida=0xa2a677
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v84 ida=0xa75e3a
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v87 ida=0xac2313
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v92 ida=0x9d64a0
// packet-audit:verify packet=cash/clientbound/CashCancelNameChangeResult version=gms_v95 ida=0xa01b10
func TestCancelNameChangeResultRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		result  byte
		message string
	}{
		{"cancelled", 0x00, ""},
		{"cancelled_alt", 0xFF, ""},
		{"failed_with_message", 0x01, "your name change request could not be processed"},
		{"failed_no_message", 0x02, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range cnrFixtureVersions {
				t.Run(v.name, func(t *testing.T) {
					ctx := pt.CreateContext(v.region, v.major, v.minor)
					input := NewCancelNameChangeResult(tc.result, tc.message)
					output := CancelNameChangeResult{}
					pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

					if output.Result() != input.Result() {
						t.Errorf("result: got %d, want %d", output.Result(), input.Result())
					}
					if output.HasMessage() != input.HasMessage() {
						t.Errorf("hasMessage: got %t, want %t", output.HasMessage(), input.HasMessage())
					}
					if output.Message() != input.Message() {
						t.Errorf("message: got %q, want %q", output.Message(), input.Message())
					}
				})
			}
		})
	}
}

func TestCancelNameChangeResultOperation(t *testing.T) {
	m := CancelNameChangeResult{}
	if m.Operation() != CashShopCancelNameChangeResultWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), CashShopCancelNameChangeResultWriter)
	}
	if CashShopCancelNameChangeResultWriter != "CashShopCancelNameChangeResult" {
		t.Errorf("writer name = %q", CashShopCancelNameChangeResultWriter)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// The round-trip test above is SYMMETRIC: a change to the shared framing
// moves encode and decode together and stays green while every real client
// read desyncs. The literal byte arrays below are written independently of
// this codec's own Encode output and pin the exact bytes.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteByte: the raw byte.
//   - WriteBool: 0x00/0x01.
//   - WriteAsciiString: uint16 LE length prefix + raw ASCII bytes.
//   - libs/atlas-socket/request/reader.go mirrors all three.
//
// Field order: derivation.md §2.6, re-confirmed by direct decompile of v61,
// v83, v92 and v95 during Task 20 — all four byte-for-byte identical in
// shape. The three shapes:
//
//	result 0x00                                       -> 00
//	result 0xFF                                        -> FF
//	result 0x01, hasMessage=true,  message "Hi" (0x0048 0x69) -> 01 01 02 00 48 69
//	result 0x02, hasMessage=false                       -> 02 00
// ---------------------------------------------------------------------------

type cnrFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
}

// The eight cells this row claims.
var cnrFixtureVersions = []cnrFixtureVersion{
	{"gms_v61", "GMS", 61, 1},
	{"gms_v72", "GMS", 72, 1},
	{"gms_v79", "GMS", 79, 1},
	{"gms_v83", "GMS", 83, 1},
	{"gms_v84", "GMS", 84, 1},
	{"gms_v87", "GMS", 87, 1},
	{"gms_v92", "GMS", 92, 1},
	{"gms_v95", "GMS", 95, 1},
}

func decodeCancelNameChangeResultBytes(t *testing.T, name string, region string, major uint16, minor uint16, body []byte) CancelNameChangeResult {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(region, major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CancelNameChangeResult
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

func TestCancelNameChangeResultByteFixtureCancelled(t *testing.T) {
	want := []byte{0x00}
	for _, v := range cnrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelNameChangeResult(0x00, "")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelNameChangeResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x00 {
				t.Errorf("%s result: got %d, want 0", v.name, out.Result())
			}
			if out.HasMessage() {
				t.Errorf("%s: unexpected hasMessage on the cancelled arm", v.name)
			}
		})
	}
}

func TestCancelNameChangeResultByteFixtureCancelledAlt(t *testing.T) {
	want := []byte{0xFF}
	for _, v := range cnrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelNameChangeResult(0xFF, "")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelNameChangeResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0xFF {
				t.Errorf("%s result: got %d, want 255", v.name, out.Result())
			}
			if out.HasMessage() {
				t.Errorf("%s: unexpected hasMessage on the cancelled_alt arm", v.name)
			}
		})
	}
}

func TestCancelNameChangeResultByteFixtureFailedWithMessage(t *testing.T) {
	want := []byte{0x01, 0x01, 0x02, 0x00, 'H', 'i'}
	for _, v := range cnrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelNameChangeResult(0x01, "Hi")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelNameChangeResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x01 {
				t.Errorf("%s result: got %d, want 1", v.name, out.Result())
			}
			if !out.HasMessage() {
				t.Errorf("%s: expected hasMessage true", v.name)
			}
			if out.Message() != "Hi" {
				t.Errorf("%s message: got %q, want %q", v.name, out.Message(), "Hi")
			}
		})
	}
}

func TestCancelNameChangeResultByteFixtureFailedNoMessage(t *testing.T) {
	want := []byte{0x02, 0x00}
	for _, v := range cnrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelNameChangeResult(0x02, "")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelNameChangeResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x02 {
				t.Errorf("%s result: got %d, want 2", v.name, out.Result())
			}
			if out.HasMessage() {
				t.Errorf("%s: unexpected hasMessage", v.name)
			}
		})
	}
}

// TestCancelNameChangeResultNoVersionDivergence pins the "identical on every
// applicable version" claim explicitly, so a future version gate added to
// this codec cannot pass silently. It also covers the v84 off-by-one trap:
// v84 must be byte-identical to v83 here.
func TestCancelNameChangeResultNoVersionDivergence(t *testing.T) {
	m := NewCancelNameChangeResult(0x01, "divergence-check")
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range cnrFixtureVersions {
		got := pt.Encode(t, pt.CreateContext(v.region, v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// ---------------------------------------------------------------------------
// Config-resolved codes (DOM-25).
// ---------------------------------------------------------------------------

func cnrOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CancelNameChangeResultCancelled:    float64(0x00),
			CancelNameChangeResultCancelledAlt: float64(0xFF),
			CancelNameChangeResultFailed:       float64(0x01),
		},
	}
}

func TestCancelNameChangeResultCodesAreConfigResolved(t *testing.T) {
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelNameChangeResultCancelledBody(), cnrOptions())
	if got[0] != 0x00 {
		t.Errorf("cancelled resolved code = %d, want 0", got[0])
	}
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelNameChangeResultCancelledAltBody(), cnrOptions())
	if got[0] != 0xFF {
		t.Errorf("cancelled_alt resolved code = %d, want 255", got[0])
	}
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelNameChangeResultFailedBody("oops"), cnrOptions())
	if got[0] != 0x01 {
		t.Errorf("failed resolved code = %d, want 1", got[0])
	}

	// Re-resolving the SAME key against a template that configures a
	// different byte must follow the config, not a Go constant.
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			CancelNameChangeResultFailed: float64(9),
		},
	}
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelNameChangeResultFailedBody(""), remapped)
	if got[0] != 9 {
		t.Errorf("remapped code = %d, want 9 — the byte is not config-resolved", got[0])
	}
}

func TestCancelNameChangeResultFailedBodyEmptyMessage(t *testing.T) {
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelNameChangeResultFailedBody(""), cnrOptions())
	want := []byte{0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("empty-message failed body:\n got %#v\nwant %#v", got, want)
	}
}
