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
// CANCEL_TRANSFER_WORLD_RESULT (clientbound) — CWvsContext::OnCancelTransferWorldResult.
//
// EIGHT applicable cells. Absent on gms_v48 and jms_v185, both confirmed by
// direct func_query against the respective IDBs (no "*OnCancelTransferWorldResult*"
// and no "*CUICancelCharacterCoupon*" match on either) — see the type doc
// comment for the positive-evidence detail.
//
// This is a NOTIFICATION: no version has a serverbound counterpart. Decode is
// carried anyway, per the standing rule, so the fixture test proves
// byte-exactness independent of Encode.
//
// The `ida=` address on each marker is that version's RECEIVER, read out of
// the version's own IDB. v61, v83, v92 and v95 were directly re-decompiled
// this pass; v72, v79, v84 and v87 were also directly re-decompiled this
// pass and confirmed structurally identical (same three-way shape switch,
// same 0x00/0x01 success sentinels):
//
//	gms_v61  0x84ae56   gms_v72  0x92254f   gms_v79  0x974684
//	gms_v83  0xa2a82d   gms_v84  0xa75ff0   gms_v87  0xac24c9
//	gms_v92  0x9d6680   gms_v95  0xa01cf0
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v61 ida=0x84ae56
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v72 ida=0x92254f
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v79 ida=0x974684
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v83 ida=0xa2a82d
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v84 ida=0xa75ff0
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v87 ida=0xac24c9
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v92 ida=0x9d6680
// packet-audit:verify packet=cash/clientbound/CashCancelTransferWorldResult version=gms_v95 ida=0xa01cf0
func TestCancelTransferWorldResultRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		result  byte
		message string
	}{
		{"cancelled", 0x00, ""},
		{"cancelled_alt", 0x01, ""},
		{"failed_with_message", 0x02, "your world transfer request could not be processed"},
		{"failed_no_message", 0x03, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range ctwrFixtureVersions {
				t.Run(v.name, func(t *testing.T) {
					ctx := pt.CreateContext(v.region, v.major, v.minor)
					input := NewCancelTransferWorldResult(tc.result, tc.message)
					output := CancelTransferWorldResult{}
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

func TestCancelTransferWorldResultOperation(t *testing.T) {
	m := CancelTransferWorldResult{}
	if m.Operation() != CashShopCancelTransferWorldResultWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), CashShopCancelTransferWorldResultWriter)
	}
	if CashShopCancelTransferWorldResultWriter != "CashShopCancelTransferWorldResult" {
		t.Errorf("writer name = %q", CashShopCancelTransferWorldResultWriter)
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
// Field order: derivation.md §2.7, re-confirmed by direct decompile of v61,
// v83, v92 and v95 this pass — all four byte-for-byte identical in shape.
// The three shapes (NOTE the 0x01 success sentinel here, vs 0xFF on the
// sibling CANCEL_NAME_CHANGE_RESULT — derivation.md §2.7's asymmetry note):
//
//	result 0x00                                                -> 00
//	result 0x01                                                -> 01
//	result 0x02, hasMessage=true,  message "Hi" (0x0048 0x69)  -> 02 01 02 00 48 69
//	result 0x03, hasMessage=false                              -> 03 00
// ---------------------------------------------------------------------------

type ctwrFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
}

// The eight cells this row claims.
var ctwrFixtureVersions = []ctwrFixtureVersion{
	{"gms_v61", "GMS", 61, 1},
	{"gms_v72", "GMS", 72, 1},
	{"gms_v79", "GMS", 79, 1},
	{"gms_v83", "GMS", 83, 1},
	{"gms_v84", "GMS", 84, 1},
	{"gms_v87", "GMS", 87, 1},
	{"gms_v92", "GMS", 92, 1},
	{"gms_v95", "GMS", 95, 1},
}

func decodeCancelTransferWorldResultBytes(t *testing.T, name string, region string, major uint16, minor uint16, body []byte) CancelTransferWorldResult {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(region, major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CancelTransferWorldResult
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

func TestCancelTransferWorldResultByteFixtureCancelled(t *testing.T) {
	want := []byte{0x00}
	for _, v := range ctwrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelTransferWorldResult(0x00, "")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelTransferWorldResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x00 {
				t.Errorf("%s result: got %d, want 0", v.name, out.Result())
			}
			if out.HasMessage() {
				t.Errorf("%s: unexpected hasMessage on the cancelled arm", v.name)
			}
		})
	}
}

func TestCancelTransferWorldResultByteFixtureCancelledAlt(t *testing.T) {
	want := []byte{0x01}
	for _, v := range ctwrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelTransferWorldResult(0x01, "")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelTransferWorldResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x01 {
				t.Errorf("%s result: got %d, want 1", v.name, out.Result())
			}
			if out.HasMessage() {
				t.Errorf("%s: unexpected hasMessage on the cancelled_alt arm", v.name)
			}
		})
	}
}

func TestCancelTransferWorldResultByteFixtureFailedWithMessage(t *testing.T) {
	want := []byte{0x02, 0x01, 0x02, 0x00, 'H', 'i'}
	for _, v := range ctwrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelTransferWorldResult(0x02, "Hi")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelTransferWorldResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x02 {
				t.Errorf("%s result: got %d, want 2", v.name, out.Result())
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

func TestCancelTransferWorldResultByteFixtureFailedNoMessage(t *testing.T) {
	want := []byte{0x03, 0x00}
	for _, v := range ctwrFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCancelTransferWorldResult(0x03, "")
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			out := decodeCancelTransferWorldResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.Result() != 0x03 {
				t.Errorf("%s result: got %d, want 3", v.name, out.Result())
			}
			if out.HasMessage() {
				t.Errorf("%s: unexpected hasMessage", v.name)
			}
		})
	}
}

// TestCancelTransferWorldResultNoVersionDivergence pins the "identical on
// every applicable version" claim explicitly, so a future version gate added
// to this codec cannot pass silently. It also covers the v84 off-by-one
// trap: v84 must be byte-identical to v83 here.
func TestCancelTransferWorldResultNoVersionDivergence(t *testing.T) {
	m := NewCancelTransferWorldResult(0x02, "divergence-check")
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range ctwrFixtureVersions {
		got := pt.Encode(t, pt.CreateContext(v.region, v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// TestCancelTransferWorldResultZeroFFAsymmetry pins the derivation.md §2.7
// asymmetry against the sibling CANCEL_NAME_CHANGE_RESULT directly: 0xFF is
// NOT a success sentinel here — it must decode as the message-carrying
// failure shape, not as CancelledAlt.
func TestCancelTransferWorldResultZeroFFAsymmetry(t *testing.T) {
	if !cancelTransferWorldResultIsShapeSwitch(0xFF) {
		t.Fatal("0xFF must be treated as the failure shape on CANCEL_TRANSFER_WORLD_RESULT (its success sentinel is 0x01, not 0xFF)")
	}
	if cancelTransferWorldResultIsShapeSwitch(0x01) {
		t.Fatal("0x01 must be a success sentinel on CANCEL_TRANSFER_WORLD_RESULT")
	}
}

// ---------------------------------------------------------------------------
// Config-resolved codes (DOM-25).
// ---------------------------------------------------------------------------

func ctwrOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CancelTransferWorldResultCancelled:    float64(0x00),
			CancelTransferWorldResultCancelledAlt: float64(0x01),
			CancelTransferWorldResultFailed:       float64(0x02),
		},
	}
}

func TestCancelTransferWorldResultCodesAreConfigResolved(t *testing.T) {
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelTransferWorldResultCancelledBody(), ctwrOptions())
	if got[0] != 0x00 {
		t.Errorf("cancelled resolved code = %d, want 0", got[0])
	}
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelTransferWorldResultCancelledAltBody(), ctwrOptions())
	if got[0] != 0x01 {
		t.Errorf("cancelled_alt resolved code = %d, want 1", got[0])
	}
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelTransferWorldResultFailedBody("oops"), ctwrOptions())
	if got[0] != 0x02 {
		t.Errorf("failed resolved code = %d, want 2", got[0])
	}

	// Re-resolving the SAME key against a template that configures a
	// different byte must follow the config, not a Go constant.
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			CancelTransferWorldResultFailed: float64(9),
		},
	}
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelTransferWorldResultFailedBody(""), remapped)
	if got[0] != 9 {
		t.Errorf("remapped code = %d, want 9 — the byte is not config-resolved", got[0])
	}
}

func TestCancelTransferWorldResultFailedBodyEmptyMessage(t *testing.T) {
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), CancelTransferWorldResultFailedBody(""), ctwrOptions())
	want := []byte{0x02, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("empty-message failed body:\n got %#v\nwant %#v", got, want)
	}
}
