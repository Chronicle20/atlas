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
// CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT (clientbound) —
// CCashShop::OnCheckTransferWorldPossibleResult.
//
// TEN applicable cells: all nine GMS versions plus jms_v185. The `ida=`
// address on each marker is that version's RECEIVER, independently
// re-decompiled during this task:
//
//	gms_v48  0x455d25   gms_v61  0x463ba6   gms_v72  0x4737ca
//	gms_v79  0x474c96   gms_v83  0x47bd9b   gms_v84  0x47ef39
//	gms_v87  0x48757c   gms_v92  0x494040   gms_v95  0x4980b0
//	jms_v185 0x48e7a6
//
// Every GMS receiver reads the same six calls in the same order — Decode4
// (discarded), Decode1, Decode4, Decode1, guarded Decode4, guarded DecodeStr.
// jms_v185 drops the third call (nBirthDate) entirely.
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v48 ida=0x455d25
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v61 ida=0x463ba6
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v72 ida=0x4737ca
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v79 ida=0x474c96
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v83 ida=0x47bd9b
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v84 ida=0x47ef39
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v87 ida=0x48757c
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v92 ida=0x494040
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=gms_v95 ida=0x4980b0
// packet-audit:verify packet=cash/clientbound/CashCheckTransferWorldPossibleResult version=jms_v185 ida=0x48e7a6
func TestCheckTransferWorldPossibleResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckTransferWorldPossibleResult(123456, 2, 19900102, []string{"Scania", "Bera"})
			output := CheckTransferWorldPossibleResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %d, want %d", output.CharacterId(), input.CharacterId())
			}
			if output.Result() != input.Result() {
				t.Errorf("result: got %d, want %d", output.Result(), input.Result())
			}
			if v.Region != "JMS" && output.BirthDate() != input.BirthDate() {
				t.Errorf("birthDate: got %d, want %d", output.BirthDate(), input.BirthDate())
			}
			if v.Region == "JMS" && output.BirthDate() != 0 {
				t.Errorf("jms birthDate: got %d, want 0 (field not on the wire)", output.BirthDate())
			}
			if output.HasWorldList() != input.HasWorldList() {
				t.Errorf("hasWorldList: got %v, want %v", output.HasWorldList(), input.HasWorldList())
			}
			if len(output.WorldNames()) != len(input.WorldNames()) {
				t.Fatalf("worldNames: got %v, want %v", output.WorldNames(), input.WorldNames())
			}
			for i := range input.WorldNames() {
				if output.WorldNames()[i] != input.WorldNames()[i] {
					t.Errorf("worldNames[%d]: got %q, want %q", i, output.WorldNames()[i], input.WorldNames()[i])
				}
			}
		})
	}
}

// TestCheckTransferWorldPossibleResultRoundTripNoWorldList covers the
// bHasWorldList=false branch, which skips the count + loop entirely.
func TestCheckTransferWorldPossibleResultRoundTripNoWorldList(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckTransferWorldPossibleResult(123456, 1, 19900102, nil)
			output := CheckTransferWorldPossibleResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.HasWorldList() {
				t.Errorf("hasWorldList: got true, want false")
			}
			if len(output.WorldNames()) != 0 {
				t.Errorf("worldNames: got %v, want empty", output.WorldNames())
			}
		})
	}
}

func TestCheckTransferWorldPossibleResultOperation(t *testing.T) {
	m := CheckTransferWorldPossibleResult{}
	if m.Operation() != CashShopCheckTransferWorldPossibleResultWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), CashShopCheckTransferWorldPossibleResultWriter)
	}
	if CashShopCheckTransferWorldPossibleResultWriter != "CashShopCheckTransferWorldPossibleResult" {
		t.Errorf("writer name = %q", CashShopCheckTransferWorldPossibleResultWriter)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// The round-trip tests above are SYMMETRIC. The literal byte arrays below are
// written independently of this codec's own Encode output and pin the exact
// bytes, per version, hand-derived from each version's decompiled receiver.
//
// Framing, read out of the shared codec rather than assumed:
//   - libs/atlas-socket/response/writer.go WriteInt: binary.LittleEndian uint32.
//   - WriteByte: the raw byte. WriteBool: 1 or 0 as a single byte.
//   - WriteAsciiString: uint16 LE length prefix + raw (Shift-JIS-encoded, but
//     ASCII text round-trips byte-identical) bytes.
//   - libs/atlas-socket/request/reader.go mirrors all of the above.
//
// Field order: derivation.md §2.5, re-confirmed by direct decompile of all
// ten receivers during this task (v72's export entry was a stub —
// "decompilation failed; hand-trace" — and was replaced with a fresh
// decompile this pass).
//
//	characterId 0x0001E240 (123456)   -> 40 E2 01 00
//	result      0x02                  -> 02
//	birthDate   0x012FA6C6 (19900102) -> C6 A6 2F 01   (ABSENT on jms_v185)
//	hasWorldList = false              -> 00
// ---------------------------------------------------------------------------

const (
	twrFixtureCharacterId = uint32(123456)
	twrFixtureResult      = byte(2)
	twrFixtureBirthDate   = uint32(19900102)
)

var (
	twrWireCharacterId = []byte{0x40, 0xE2, 0x01, 0x00}
	twrWireResult      = []byte{0x02}
	twrWireBirthDate   = []byte{0xC6, 0xA6, 0x2F, 0x01}
	twrWireNoWorldList = []byte{0x00}
)

type transferWorldResultFixtureVersion struct {
	name     string
	region   string
	major    uint16
	minor    uint16
	hasBirth bool
}

// The ten cells this row claims.
var transferWorldResultFixtureVersions = []transferWorldResultFixtureVersion{
	{"gms_v48", "GMS", 48, 1, true},
	{"gms_v61", "GMS", 61, 1, true},
	{"gms_v72", "GMS", 72, 1, true},
	{"gms_v79", "GMS", 79, 1, true},
	{"gms_v83", "GMS", 83, 1, true},
	{"gms_v84", "GMS", 84, 1, true},
	{"gms_v87", "GMS", 87, 1, true},
	{"gms_v92", "GMS", 92, 1, true},
	{"gms_v95", "GMS", 95, 1, true},
	{"jms_v185", "JMS", 185, 1, false},
}

func decodeTransferWorldResultBytes(t *testing.T, name string, region string, major uint16, minor uint16, body []byte) CheckTransferWorldPossibleResult {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext(region, major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out CheckTransferWorldPossibleResult
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

// TestCheckTransferWorldPossibleResultByteFixtureNoWorldList pins the
// bHasWorldList=false shape, which every GMS version shares (14 bytes:
// Decode4+Decode1+Decode4+Decode1) and jms_v185 shortens by the missing
// nBirthDate field (6 bytes: Decode4+Decode1+Decode1).
func TestCheckTransferWorldPossibleResultByteFixtureNoWorldList(t *testing.T) {
	for _, v := range transferWorldResultFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := []byte{}
			want = append(want, twrWireCharacterId...)
			want = append(want, twrWireResult...)
			if v.hasBirth {
				want = append(want, twrWireBirthDate...)
			}
			want = append(want, twrWireNoWorldList...)

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCheckTransferWorldPossibleResult(twrFixtureCharacterId, twrFixtureResult, twrFixtureBirthDate, nil)
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}
			wantLen := 10
			if !v.hasBirth {
				wantLen = 6
			}
			if len(got) != wantLen {
				t.Errorf("%s body = %d bytes, want %d", v.name, len(got), wantLen)
			}

			out := decodeTransferWorldResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if out.CharacterId() != twrFixtureCharacterId {
				t.Errorf("%s characterId: got %d, want %d", v.name, out.CharacterId(), twrFixtureCharacterId)
			}
			if out.Result() != twrFixtureResult {
				t.Errorf("%s result: got %d, want %d", v.name, out.Result(), twrFixtureResult)
			}
			wantBirth := twrFixtureBirthDate
			if !v.hasBirth {
				wantBirth = 0
			}
			if out.BirthDate() != wantBirth {
				t.Errorf("%s birthDate: got %d, want %d", v.name, out.BirthDate(), wantBirth)
			}
			if out.HasWorldList() {
				t.Errorf("%s hasWorldList: got true, want false", v.name)
			}
		})
	}
}

// TestCheckTransferWorldPossibleResultByteFixtureWithWorldList pins the
// bHasWorldList=true shape with a two-entry world-name list ("Scania","Bera"),
// hand-encoded per libs/atlas-socket/response/writer.go WriteAsciiString
// (uint16 LE length + raw bytes) and WriteInt (uint32 LE count).
func TestCheckTransferWorldPossibleResultByteFixtureWithWorldList(t *testing.T) {
	worldNames := []string{"Scania", "Bera"}
	wireWorldList := []byte{
		0x01,                   // hasWorldList = true
		0x02, 0x00, 0x00, 0x00, // count = 2
		0x06, 0x00, 'S', 'c', 'a', 'n', 'i', 'a', // "Scania"
		0x04, 0x00, 'B', 'e', 'r', 'a', // "Bera"
	}

	for _, v := range transferWorldResultFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			want := []byte{}
			want = append(want, twrWireCharacterId...)
			want = append(want, byte(0)) // ALLOWED arm
			if v.hasBirth {
				want = append(want, twrWireBirthDate...)
			}
			want = append(want, wireWorldList...)

			ctx := pt.CreateContext(v.region, v.major, v.minor)
			in := NewCheckTransferWorldPossibleResult(twrFixtureCharacterId, 0, twrFixtureBirthDate, worldNames)
			got := pt.Encode(t, ctx, in.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s encode:\n got %#v\nwant %#v", v.name, got, want)
			}

			out := decodeTransferWorldResultBytes(t, v.name, v.region, v.major, v.minor, want)
			if !out.HasWorldList() {
				t.Fatalf("%s hasWorldList: got false, want true", v.name)
			}
			if len(out.WorldNames()) != len(worldNames) {
				t.Fatalf("%s worldNames: got %v, want %v", v.name, out.WorldNames(), worldNames)
			}
			for i, name := range worldNames {
				if out.WorldNames()[i] != name {
					t.Errorf("%s worldNames[%d]: got %q, want %q", v.name, i, out.WorldNames()[i], name)
				}
			}
		})
	}
}

// TestCheckTransferWorldPossibleResultGMSNoVersionDivergence pins the "no
// field/width/order divergence across the nine GMS versions" claim, so a
// future version gate cannot pass silently. It also covers the v84 off-by-one
// trap: v84 must be byte-identical to v83 here.
func TestCheckTransferWorldPossibleResultGMSNoVersionDivergence(t *testing.T) {
	m := NewCheckTransferWorldPossibleResult(twrFixtureCharacterId, twrFixtureResult, twrFixtureBirthDate, nil)
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range transferWorldResultFixtureVersions {
		if v.region != "GMS" {
			continue
		}
		got := pt.Encode(t, pt.CreateContext(v.region, v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// TestCheckTransferWorldPossibleResultJmsDropsBirthDate pins the one
// structural divergence this row has: jms_v185 is 4 bytes shorter than every
// GMS version because its receiver (@0x48e7a6) never calls the third Decode4
// (nBirthDate) that every GMS receiver does.
func TestCheckTransferWorldPossibleResultJmsDropsBirthDate(t *testing.T) {
	m := NewCheckTransferWorldPossibleResult(twrFixtureCharacterId, twrFixtureResult, twrFixtureBirthDate, nil)
	gms := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	jms := pt.Encode(t, pt.CreateContext("JMS", 185, 1), m.Encode, nil)
	if len(gms)-len(jms) != 4 {
		t.Fatalf("gms/jms length delta = %d, want 4 (the dropped nBirthDate)", len(gms)-len(jms))
	}
}

// ---------------------------------------------------------------------------
// Config-resolved reason codes (DOM-25).
// ---------------------------------------------------------------------------

// twrOptions mirrors the operations table the ten seed templates carry for
// the CashShopCheckTransferWorldPossibleResult writer.
func twrOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CheckTransferWorldPossibleAllowed:           float64(0),
			CheckTransferWorldPossibleCharacterNotFound: float64(1),
			CheckTransferWorldPossibleReason2:           float64(2),
			CheckTransferWorldPossibleReason3:           float64(3),
			CheckTransferWorldPossibleReason4:           float64(4),
			CheckTransferWorldPossibleReason5:           float64(5),
			CheckTransferWorldPossibleReason6:           float64(6),
			CheckTransferWorldPossibleReason7:           float64(7),
			CheckTransferWorldPossibleInFamily:          float64(8),
			CheckTransferWorldPossibleUnknownError:      float64(9),
		},
	}
}

func TestCheckTransferWorldPossibleResultCodesAreConfigResolved(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want byte
	}{
		{CheckTransferWorldPossibleAllowed, 0},
		{CheckTransferWorldPossibleCharacterNotFound, 1},
		{CheckTransferWorldPossibleReason2, 2},
		{CheckTransferWorldPossibleReason3, 3},
		{CheckTransferWorldPossibleReason4, 4},
		{CheckTransferWorldPossibleReason5, 5},
		{CheckTransferWorldPossibleReason6, 6},
		{CheckTransferWorldPossibleReason7, 7},
		{CheckTransferWorldPossibleInFamily, 8},
		{CheckTransferWorldPossibleUnknownError, 9},
	} {
		t.Run(tc.key, func(t *testing.T) {
			body := CheckTransferWorldPossibleResultBody(twrFixtureCharacterId, tc.key, twrFixtureBirthDate, nil)
			got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, twrOptions())
			if got[4] != tc.want {
				t.Errorf("resolved code = %d, want %d", got[4], tc.want)
			}
		})
	}

	// Re-resolving the SAME key against a template that configures a
	// different byte must follow the config, not a Go constant.
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			CheckTransferWorldPossibleUnknownError: float64(42),
		},
	}
	body := CheckTransferWorldPossibleResultBody(twrFixtureCharacterId, CheckTransferWorldPossibleUnknownError, 0, nil)
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, remapped)
	if got[4] != 42 {
		t.Errorf("remapped code = %d, want 42 — the byte is not config-resolved", got[4])
	}
}

func TestCheckTransferWorldPossibleResultAllowedBody(t *testing.T) {
	body := CheckTransferWorldPossibleResultAllowedBody(twrFixtureCharacterId, twrFixtureBirthDate, nil)
	got := pt.Encode(t, pt.CreateContext("GMS", 95, 1), body, twrOptions())

	want := []byte{}
	want = append(want, twrWireCharacterId...)
	want = append(want, 0x00)
	want = append(want, twrWireBirthDate...)
	want = append(want, twrWireNoWorldList...)
	if !bytes.Equal(got, want) {
		t.Fatalf("allowed body:\n got %#v\nwant %#v", got, want)
	}
}

// TestCheckTransferWorldPossibleResultReasonMapping pins the brief's reason ->
// wire-code mapping. Only in_family lands on a distinct arm (arm 8, the one
// rejection arm with independently confirmed rendered text — see the doc
// comment on CheckTransferWorldPossibleResultRejectedBody); every other
// reason, INCLUDING is_guild_master and is_gm, collapses to UNKNOWN_ERROR.
func TestCheckTransferWorldPossibleResultReasonMapping(t *testing.T) {
	required := []string{
		"world_same", "world_unknown", "world_full", "no_character_slot",
		"banned", "is_guild_master", "is_gm", "in_family", "trade_open",
		"merchant_open", "mts_listings_open", "name_taken", "check_unavailable",
	}

	if len(checkTransferWorldPossibleReasonArms) != len(required) {
		t.Fatalf("reason table has %d entries, want %d — the taxonomy is closed", len(checkTransferWorldPossibleReasonArms), len(required))
	}
	for _, reason := range required {
		key, ok := checkTransferWorldPossibleReasonArms[reason]
		if !ok {
			t.Fatalf("reason %q missing from the mapping table", reason)
		}

		wantKey := CheckTransferWorldPossibleUnknownError
		wantCode := byte(9)
		if reason == "in_family" {
			wantKey = CheckTransferWorldPossibleInFamily
			wantCode = 8
		}
		if key != wantKey {
			t.Errorf("reason %q maps to %q, want %q", reason, key, wantKey)
		}

		body := CheckTransferWorldPossibleResultRejectedBody(twrFixtureCharacterId, reason, nil)
		got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, twrOptions())
		if got[4] != wantCode {
			t.Errorf("%s resolved code = %d, want %d", reason, got[4], wantCode)
		}
	}

	// A reason outside the closed taxonomy is a server bug; it must still
	// produce the safe unknown-error arm rather than a guessed byte.
	body := CheckTransferWorldPossibleResultRejectedBody(twrFixtureCharacterId, "not_a_reason", nil)
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, twrOptions())
	if got[4] != 9 {
		t.Errorf("unknown reason resolved code = %d, want 9", got[4])
	}
}
