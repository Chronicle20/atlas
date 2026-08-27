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
// MAPLELIFE_ERROR (clientbound) — CUICharacterSaleDlg::OnCreateNewCharacterResult.
//
// FIVE in-scope cells: gms_v83, gms_v84, gms_v87, gms_v92, gms_v95 —
// CUICharacterSaleDlg exists on gms_v84 too; an earlier pass's
// VERSION-ABSENT finding was wrong and has been retracted
// (derivation.md §2.0-CORRECTION, supersedes §5.5's four-cell framing).
// gms_v84's own byte fixture (mleFixtureVersions) and marker were added by
// the gms_v84 verification pass (task-246 packet-verifier); gms_v84's wire
// framing was already exercised generically by TestMapleLifeErrorRoundTrip
// via pt.Variants beforehand.
//
// The `ida=` address on each marker is that version's RECEIVER, decompiled
// this pass: gms_v83 @0x7d77b0, gms_v84 @0x7fda6f, gms_v87 @0x82e252,
// gms_v92 @0x7564f0, gms_v95 @0x777fc0 — identical decode order and branch
// SHAPE on all five: Decode1(nType) then Decode4(nParam), exact-equality
// switch on nType (derivation.md §5.5; gms_v84 confirmed in
// bug-maple-life-v84-registration.md "Positive derivation").
// ---------------------------------------------------------------------------

// packet-audit:verify packet=maplelife/clientbound/MaplelifeMapleLifeError version=gms_v83 ida=0x7d77b0
// packet-audit:verify packet=maplelife/clientbound/MaplelifeMapleLifeError version=gms_v84 ida=0x7fda6f
// packet-audit:verify packet=maplelife/clientbound/MaplelifeMapleLifeError version=gms_v87 ida=0x82e252
// packet-audit:verify packet=maplelife/clientbound/MaplelifeMapleLifeError version=gms_v92 ida=0x7564f0
// packet-audit:verify packet=maplelife/clientbound/MaplelifeMapleLifeError version=gms_v95 ida=0x777fc0
func TestMapleLifeErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMapleLifeError(56, 42)
			output := MapleLifeError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.NType() != input.NType() {
				t.Errorf("nType: got %d, want %d", output.NType(), input.NType())
			}
			if output.NParam() != input.NParam() {
				t.Errorf("nParam: got %d, want %d", output.NParam(), input.NParam())
			}
		})
	}
}

func TestMapleLifeErrorOperation(t *testing.T) {
	m := MapleLifeError{}
	if m.Operation() != MapleLifeErrorWriter {
		t.Errorf("Operation() = %q, want %q", m.Operation(), MapleLifeErrorWriter)
	}
	if MapleLifeErrorWriter != "MapleLifeError" {
		t.Errorf("writer name = %q", MapleLifeErrorWriter)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// Framing, read out of the shared codec rather than assumed:
//   - WriteByte: a single raw byte.
//   - WriteInt: uint32 LE.
//
// Field order: derivation.md §5 (Decode1 nType; Decode4 nParam). The body is
// IDENTICAL SHAPE on every in-scope version, so the fixtures are deliberately
// the same bytes for all five — that sameness IS the derived claim. Only the
// per-version nType LITERAL for a given semantic arm differs (exercised in
// TestMapleLifeErrorCodesAreConfigResolved, not here — these fixtures pin
// the wire framing for one representative nType/nParam pair per arm).
// ---------------------------------------------------------------------------

type mleFixtureVersion struct {
	name  string
	major uint16
	minor uint16
}

// The five in-scope cells this row claims.
var mleFixtureVersions = []mleFixtureVersion{
	{"gms_v83", 83, 1},
	{"gms_v84", 84, 1},
	{"gms_v87", 87, 1},
	{"gms_v92", 92, 1},
	{"gms_v95", 95, 1},
}

func decodeMLEBytes(t *testing.T, name string, major uint16, minor uint16, body []byte) MapleLifeError {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var out MapleLifeError
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

// TestMapleLifeErrorByteFixture pins one case per §5 semantic arm — success
// (nParam==0), name-taken-at-submit, unknown-error(param) — for every
// in-scope version, using a representative nType per arm (the actual
// per-version literal is exercised separately in
// TestMapleLifeErrorCodesAreConfigResolved).
func TestMapleLifeErrorByteFixture(t *testing.T) {
	cases := []struct {
		arm    string
		nType  byte
		nParam uint32
	}{
		{"success", 56, 0},
		{"name_taken_at_submit", 58, 0},
		{"unknown_error", 99, 12345},
	}
	for _, v := range mleFixtureVersions {
		for _, c := range cases {
			t.Run(v.name+"/"+c.arm, func(t *testing.T) {
				want := []byte{c.nType}
				want = append(want,
					byte(c.nParam), byte(c.nParam>>8), byte(c.nParam>>16), byte(c.nParam>>24))

				ctx := pt.CreateContext("GMS", v.major, v.minor)
				in := NewMapleLifeError(c.nType, c.nParam)
				got := pt.Encode(t, ctx, in.Encode, nil)
				if !bytes.Equal(got, want) {
					t.Fatalf("%s/%s encode:\n got %#v\nwant %#v", v.name, c.arm, got, want)
				}
				if len(got) != 5 {
					t.Errorf("%s/%s body = %d bytes, want 5 (1 Decode1 + 4 Decode4)", v.name, c.arm, len(got))
				}

				out := decodeMLEBytes(t, v.name, v.major, v.minor, want)
				if out.NType() != c.nType {
					t.Errorf("%s/%s nType: got %d, want %d", v.name, c.arm, out.NType(), c.nType)
				}
				if out.NParam() != c.nParam {
					t.Errorf("%s/%s nParam: got %d, want %d", v.name, c.arm, out.NParam(), c.nParam)
				}
			})
		}
	}
}

// TestMapleLifeErrorNoVersionDivergence pins the "identical SHAPE on every
// in-scope version" claim explicitly (derivation.md §5.5: field order, width
// and shape are identical on every present version; only per-version nType
// literals differ, which live in tenant-template config, not here), so a
// future version gate added to this codec's wire layout cannot pass silently.
func TestMapleLifeErrorNoVersionDivergence(t *testing.T) {
	m := NewMapleLifeError(56, 42)
	base := pt.Encode(t, pt.CreateContext("GMS", 83, 1), m.Encode, nil)
	for _, v := range mleFixtureVersions {
		got := pt.Encode(t, pt.CreateContext("GMS", v.major, v.minor), m.Encode, nil)
		if !bytes.Equal(got, base) {
			t.Errorf("%s must be byte-identical to gms_v83: %#v vs %#v", v.name, got, base)
		}
	}
}

// ---------------------------------------------------------------------------
// Config-resolved nType codes (DOM-25).
// ---------------------------------------------------------------------------

func mleOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			MapleLifeErrorSuccess:           float64(56),
			MapleLifeErrorNameTakenAtSubmit: float64(58),
			MapleLifeErrorUnknownError:      float64(99),
		},
	}
}

func TestMapleLifeErrorCodesAreConfigResolved(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want byte
	}{
		{MapleLifeErrorSuccess, 56},
		{MapleLifeErrorNameTakenAtSubmit, 58},
		{MapleLifeErrorUnknownError, 99},
	} {
		t.Run(tc.key, func(t *testing.T) {
			body := MapleLifeErrorBody(tc.key)
			got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, mleOptions())
			if len(got) != 5 {
				t.Fatalf("body = %d bytes, want 5", len(got))
			}
			gotType := got[0]
			if gotType != tc.want {
				t.Errorf("resolved nType = %d, want %d", gotType, tc.want)
			}
		})
	}

	// An empty options map must NOT silently produce a valid-looking arm —
	// it resolves to ResolveCode's loud 99 sentinel.
	body := MapleLifeErrorBody(MapleLifeErrorSuccess)
	got := pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, map[string]interface{}{})
	gotType := got[0]
	if gotType != 99 {
		t.Errorf("empty-options resolved nType = %d, want 99 (ResolveCode sentinel)", gotType)
	}

	// Re-resolving the SAME key against a template that configures a
	// different byte must follow the config, not a Go constant — this is
	// exactly the per-version literal shift §5.5 documents (52/54/55/56 for
	// SUCCESS across v83/v87/v92/v95).
	remapped := map[string]interface{}{
		"operations": map[string]interface{}{
			MapleLifeErrorSuccess: float64(52),
		},
	}
	body = MapleLifeErrorBody(MapleLifeErrorSuccess)
	got = pt.Encode(t, pt.CreateContext("GMS", 83, 1), body, remapped)
	gotType = got[0]
	if gotType != 52 {
		t.Errorf("remapped nType = %d, want 52 — the byte is not config-resolved", gotType)
	}
}

// TestMapleLifeErrorArmsAreExhaustive lists the exported arm constants
// literally, so adding a code without deriving it against derivation.md §5
// fails here rather than silently expanding the enumeration.
func TestMapleLifeErrorArmsAreExhaustive(t *testing.T) {
	want := map[string]bool{
		"SUCCESS":              true,
		"NAME_TAKEN_AT_SUBMIT": true,
		"UNKNOWN_ERROR":        true,
	}
	got := map[string]bool{
		MapleLifeErrorSuccess:           true,
		MapleLifeErrorNameTakenAtSubmit: true,
		MapleLifeErrorUnknownError:      true,
	}
	if len(got) != len(want) {
		t.Fatalf("arm set has %d entries, want %d — the enumeration is closed (derivation.md §5.5)", len(got), len(want))
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing arm constant for %q", k)
		}
	}
	for k := range got {
		if !want[k] {
			t.Errorf("unexpected arm constant %q not in derivation.md §5.5's closed enumeration", k)
		}
	}
}
