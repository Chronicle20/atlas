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
// CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT (clientbound) —
// CCashShop::OnCheckNameChangePossibleResult.
//
// SIX applicable cells. The op arrives at v79 (gms_v48/v61/v72 have no
// OnCheckNameChangePossibleResult receiver) and jms_v185 has no name-change
// feature at all (derivation.md §1.5).
//
// The `ida=` address on each marker is that version's RECEIVER, read out of the
// version's own IDB and re-decompiled during this task. Every one reads the same
// three fields in the same order — Decode4 (discarded), Decode1, Decode4:
//
//	gms_v79  0x474ab1   gms_v83  0x47bbb6   gms_v84  0x47ed54
//	gms_v87  0x487397   gms_v92  0x4913a0   gms_v95  0x495470
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/clientbound/CashCheckNameChangePossibleResult version=gms_v79 ida=0x474ab1
// packet-audit:verify packet=cash/clientbound/CashCheckNameChangePossibleResult version=gms_v83 ida=0x47bbb6
// packet-audit:verify packet=cash/clientbound/CashCheckNameChangePossibleResult version=gms_v84 ida=0x47ed54
// packet-audit:verify packet=cash/clientbound/CashCheckNameChangePossibleResult version=gms_v87 ida=0x487397
// packet-audit:verify packet=cash/clientbound/CashCheckNameChangePossibleResult version=gms_v92 ida=0x4913a0
// packet-audit:verify packet=cash/clientbound/CashCheckNameChangePossibleResult version=gms_v95 ida=0x495470
func TestCheckNameChangePossibleResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckNameChangePossibleResult(123456, 2, 19900102)
			output := CheckNameChangePossibleResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %d, want %d", output.CharacterId(), input.CharacterId())
			}
			if output.Result() != input.Result() {
				t.Errorf("result: got %d, want %d", output.Result(), input.Result())
			}
			if output.BirthDate() != input.BirthDate() {
				t.Errorf("birthDate: got %d, want %d", output.BirthDate(), input.BirthDate())
			}
		})
	}
}

func TestCheckNameChangePossibleResultOperation(t *testing.T) {
	m := CheckNameChangePossibleResult{}
	if m.Operation() != CashShopCheckNameChangePossibleResultWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), CashShopCheckNameChangePossibleResultWriter)
	}
	if CashShopCheckNameChangePossibleResultWriter != "CashShopCheckNameChangePossibleResult" {
		t.Errorf("writer name = %q", CashShopCheckNameChangePossibleResultWriter)
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
//   - libs/atlas-socket/response/writer.go WriteInt: binary.LittleEndian uint32.
//   - WriteByte: the raw byte.
//   - libs/atlas-socket/request/reader.go mirrors both.
//
// Field order: derivation.md §2.4, re-confirmed by decompiling all six receivers
// during this task. The body is IDENTICAL on every applicable version, so the
// six fixtures are deliberately the same nine bytes — that sameness IS the
// derived claim, and a future version gate that broke it would fail here.
//
//	characterId 0x0001E240 (123456)   -> 40 E2 01 00
//	result      0x02                  -> 02
//	birthDate   0x012FA6C6 (19900102) -> C6 A6 2F 01
// ---------------------------------------------------------------------------

const (
	nrFixtureCharacterId = uint32(123456)
	nrFixtureResult      = byte(2)
	nrFixtureBirthDate   = uint32(19900102)
)

var (
	nrWireCharacterId = []byte{0x40, 0xE2, 0x01, 0x00}
	nrWireResult      = []byte{0x02}
	nrWireBirthDate   = []byte{0xC6, 0xA6, 0x2F, 0x01}
)

type nameChangeResultFixtureVersion struct {
	name   string
	region string
	major  uint16
	minor  uint16
}

// The six cells this row claims.
var nameChangeResultFixtureVersions = []nameChangeResultFixtureVersion{
	{"gms_v79", "GMS", 79, 1},
	{"gms_v83", "GMS", 83, 1},
	{"gms_v84", "GMS", 84, 1},
	{"gms_v87", "GMS", 87, 1},
	{"gms_v92", "GMS", 92, 1},
	{"gms_v95", "GMS", 95, 1},
}

func decodeNameChangeResultBytes(t *testing.T, name string, region string, major uint16, minor uint16, body []byte) CheckNameChangePossibleResult {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(region, major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CheckNameChangePossibleResult
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

func TestCheckNameChangePossibleResultByteFixture(t *testing.T) {
	for _, v := range nameChangeResultFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := []byte{}
			want = append(want, nrWireCharacterId...)
			want = append(want, nrWireResult...)
			want = append(want, nrWireBirthDate...)

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCheckNameChangePossibleResult(nrFixtureCharacterId, nrFixtureResult, nrFixtureBirthDate)
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			if len(got) != 9 {
				t.Errorf("%s body = %d bytes, want 9 (Decode4 + Decode1 + Decode4)", v.name, len(got))
			}

			out := decodeNameChangeResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.CharacterId() != nrFixtureCharacterId {
				t.Errorf("%s characterId: got %d, want %d", v.name, out.CharacterId(), nrFixtureCharacterId)
			}
			if out.Result() != nrFixtureResult {
				t.Errorf("%s result: got %d, want %d", v.name, out.Result(), nrFixtureResult)
			}
			if out.BirthDate() != nrFixtureBirthDate {
				t.Errorf("%s birthDate: got %d, want %d", v.name, out.BirthDate(), nrFixtureBirthDate)
			}
		})
	}
}

// TestCheckNameChangePossibleResultNoVersionDivergence pins the "identical on
// every applicable version" claim explicitly, so a future version gate added to
// this codec cannot pass silently. It also covers the v84 off-by-one trap: v84
// must be byte-identical to v83 here.
func TestCheckNameChangePossibleResultNoVersionDivergence(t *testing.T) {
	m := NewCheckNameChangePossibleResult(nrFixtureCharacterId, nrFixtureResult, nrFixtureBirthDate)
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range nameChangeResultFixtureVersions {
		got := pt.Encode(t, pt.CreateContext(v.region, v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// ---------------------------------------------------------------------------
// Config-resolved reason codes (DOM-25).
//
// The result byte must come from the tenant template's operations table, never
// from a Go literal. These tests drive the body builders with a synthetic
// options map and assert the byte that lands on the wire is the CONFIGURED one.
// ---------------------------------------------------------------------------

// nrOptions mirrors the operations table the six seed templates carry for the
// CashShopCheckNameChangePossibleResult writer.
func nrOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CheckNameChangePossibleAllowed:               float64(0),
			CheckNameChangePossibleAlreadySubmitted:      float64(1),
			CheckNameChangePossibleRequestLimitRecent:    float64(2),
			CheckNameChangePossibleRequestLimitRequested: float64(3),
			CheckNameChangePossibleUnknownError:          float64(4),
		},
	}
}

func TestCheckNameChangePossibleResultCodesAreConfigResolved(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want byte
	}{
		{CheckNameChangePossibleAllowed, 0},
		{CheckNameChangePossibleAlreadySubmitted, 1},
		{CheckNameChangePossibleRequestLimitRecent, 2},
		{CheckNameChangePossibleRequestLimitRequested, 3},
		{CheckNameChangePossibleUnknownError, 4},
	} {
		t.Run(tc.key, func(t *testing.T) {
			body := CheckNameChangePossibleResultBody(nrFixtureCharacterId, tc.key, nrFixtureBirthDate)
			got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, nrOptions())
			if len(got) != 9 {
				t.Fatalf("body = %d bytes, want 9", len(got))
			}
			if got[4] != tc.want {
				t.Errorf("resolved code = %d, want %d", got[4], tc.want)
			}
		})
	}

	// Re-resolving the SAME key against a template that configures a different
	// byte must follow the config, not a Go constant.
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			CheckNameChangePossibleUnknownError: float64(9),
		},
	}
	body := CheckNameChangePossibleResultBody(nrFixtureCharacterId, CheckNameChangePossibleUnknownError, 0)
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, remapped)
	if got[4] != 9 {
		t.Errorf("remapped code = %d, want 9 — the byte is not config-resolved", got[4])
	}
}

func TestCheckNameChangePossibleResultAllowedBody(t *testing.T) {
	body := CheckNameChangePossibleResultAllowedBody(nrFixtureCharacterId, nrFixtureBirthDate)
	got := pt.Encode(t, pt.CreateContext("GMS", 95, 1), body, nrOptions())

	want := []byte{}
	want = append(want, nrWireCharacterId...)
	want = append(want, 0x00)
	want = append(want, nrWireBirthDate...)
	if !bytes.Equal(got, want) {
		t.Fatalf("allowed body:\n got %#v\nwant %#v", got, want)
	}
}

// TestCheckNameChangePossibleResultReasonMapping pins the design §6 reason →
// wire-code mapping. All four members of this row's required set land on the
// client's default (unknown-error) arm, because no arm of
// CCashShop::OnCheckNameChangePossibleResult renders a name-validity message —
// see the doc comment on CheckNameChangePossibleResultRejectedBody. Name
// validity travels on CASHSHOP_CHECK_NAME_CHANGE instead.
func TestCheckNameChangePossibleResultReasonMapping(t *testing.T) {
	required := []string{"name_taken", "name_reserved", "name_invalid_length", "name_invalid_charset"}

	if len(checkNameChangePossibleReasonArms) != len(required) {
		t.Fatalf("reason table has %d entries, want %d — the taxonomy is closed", len(checkNameChangePossibleReasonArms), len(required))
	}
	for _, reason := range required {
		key, ok := checkNameChangePossibleReasonArms[reason]
		if !ok {
			t.Fatalf("reason %q missing from the mapping table", reason)
		}
		if key != CheckNameChangePossibleUnknownError {
			t.Errorf("reason %q maps to %q, want %q", reason, key, CheckNameChangePossibleUnknownError)
		}

		body := CheckNameChangePossibleResultRejectedBody(nrFixtureCharacterId, reason)
		got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, nrOptions())
		if len(got) != 9 {
			t.Fatalf("%s body = %d bytes, want 9", reason, len(got))
		}
		if got[4] != 4 {
			t.Errorf("%s resolved code = %d, want 4 (the configured UNKNOWN_ERROR byte)", reason, got[4])
		}
	}

	// A reason outside the closed taxonomy is a server bug; it must still
	// produce the safe unknown-error arm rather than a guessed byte.
	body := CheckNameChangePossibleResultRejectedBody(nrFixtureCharacterId, "not_a_reason")
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, nrOptions())
	if got[4] != 4 {
		t.Errorf("unknown reason resolved code = %d, want 4", got[4])
	}
}
