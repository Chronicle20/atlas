package serverbound

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// ---------------------------------------------------------------------------
// ItemUseMapleLife — CUICharacterSaleDlg::SendCreateNewCharacter, the 543
// sub-body of USE_CASH_ITEM.
//
// FOUR in-scope cells: gms_v83, gms_v87, gms_v92, gms_v95. gms_v84 is
// VERSION-ABSENT — no CUICharacterSaleDlg code path exists on that binary
// (derivation.md §2.0) — no marker, no fixture.
//
// The `ida=` address on each marker is that version's SENDER, decompiled
// this pass: gms_v83 @0x7d7960, gms_v87 @0x82e402, gms_v92 @0x758770,
// gms_v95 @0x77a240.
//
// IMPORTANT DEVIATION FROM item_use_incubator.go's convention: this
// sub-body's trailing update_time write is UNCONDITIONAL on every version,
// not gated by `!updateTimeFirst`. Raw disassembly (item_use_maple_life.go's
// doc comment quotes the exact instruction addresses) confirms
// SendCreateNewCharacter writes update_time TWICE on gms_v87/v92/v95 (once
// via the shared ItemUse header's leading copy, once via this sub-body's own
// trailing copy) and ONCE on gms_v83 (trailing only, via this sub-body — the
// header has no leading copy on that version). The `updateTimeFirst`
// constructor parameter is therefore accepted for signature parity with the
// sibling sub-bodies but does not gate anything here.
// ---------------------------------------------------------------------------

// packet-audit:verify packet=cash/serverbound/CashItemUseMapleLife version=gms_v83 ida=0x7d7960
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleLife version=gms_v87 ida=0x82e402
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleLife version=gms_v92 ida=0x758770
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleLife version=gms_v95 ida=0x77a240
func TestItemUseMapleLifeRoundTrip(t *testing.T) {
	inScope := []pt.TenantVariant{
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "GMS v92", Region: "GMS", MajorVersion: 92, MinorVersion: 1},
		{Name: "GMS v95", Region: "GMS", MajorVersion: 95, MinorVersion: 1},
	}
	for _, first := range []bool{true, false} {
		for _, v := range inScope {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := ItemUseMapleLife{
					updateTimeFirst: first,
					name:            "Chronicle",
					al0:             1, al1: 2, al2: 3, al3: 4,
					gender: 0, currentClass: 100, sp: 5,
					updateTime: 123456789,
				}
				output := *NewItemUseMapleLife(first)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

				if output.Name() != input.Name() {
					t.Errorf("name = %q, want %q", output.Name(), input.Name())
				}
				if output.AL0() != input.AL0() || output.AL1() != input.AL1() || output.AL2() != input.AL2() || output.AL3() != input.AL3() {
					t.Errorf("al = [%d %d %d %d], want [%d %d %d %d]",
						output.AL0(), output.AL1(), output.AL2(), output.AL3(),
						input.AL0(), input.AL1(), input.AL2(), input.AL3())
				}
				if output.Gender() != input.Gender() {
					t.Errorf("gender = %d, want %d", output.Gender(), input.Gender())
				}
				if output.CurrentClass() != input.CurrentClass() {
					t.Errorf("currentClass = %d, want %d", output.CurrentClass(), input.CurrentClass())
				}
				if output.SP() != input.SP() {
					t.Errorf("sp = %d, want %d", output.SP(), input.SP())
				}
				// The trailing update_time is ALWAYS written/consumed by this
				// sub-body, regardless of updateTimeFirst — see the doc-comment
				// deviation note above.
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime = %d, want %d", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}

// TestItemUseMapleLifeUpdateTimeTrailing directly asserts the deviation from
// item_use_incubator.go's `!updateTimeFirst`-gated convention: on every
// in-scope version (v83 AND v87/v92/v95 alike), decoding a wire body always
// consumes a trailing update_time and UpdateTime() always returns the
// decoded value. This does NOT match the brief's originally-proposed
// "consumes only when updateTimeFirst==false" behavior — that assumption is
// what the controller addendum asked this task to re-confirm and correct.
// Raw disassembly (item_use_maple_life.go's doc comment) shows v83 has
// exactly one get_update_time/Encode4 pair (trailing, no leading copy exists
// to omit against), while v87/v92/v95 each have TWO independent pairs (one
// leading — modeled by the shared ItemUse header — and a separate one
// trailing, modeled here). This sub-body's own trailing write is therefore
// unconditional on all four versions.
func TestItemUseMapleLifeUpdateTimeTrailing(t *testing.T) {
	inScope := []pt.TenantVariant{
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "GMS v92", Region: "GMS", MajorVersion: 92, MinorVersion: 1},
		{Name: "GMS v95", Region: "GMS", MajorVersion: 95, MinorVersion: 1},
	}
	for _, first := range []bool{true, false} {
		for _, v := range inScope {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := ItemUseMapleLife{updateTimeFirst: first, name: "Chronicle", updateTime: 999}
				output := *NewItemUseMapleLife(first)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.UpdateTime() != 999 {
					t.Errorf("UpdateTime() = %d, want 999 (unconditional trailing write, updateTimeFirst=%v)", output.UpdateTime(), first)
				}
			})
		}
	}
}

func TestItemUseMapleLifeOperation(t *testing.T) {
	m := ItemUseMapleLife{}
	if m.Operation() != "ItemUseMapleLife" {
		t.Errorf("Operation() = %q, want %q", m.Operation(), "ItemUseMapleLife")
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// Framing, read out of the shared codec:
//   - WriteAsciiString: uint16 LE length prefix + raw bytes.
//   - WriteInt32/WriteInt: int32/uint32 LE.
//
// Field order: derivation.md §2 (sName, al[0..3], nGender, nCurrentClass,
// nSP, update_time). IDENTICAL SHAPE (field set, order, width) on every
// in-scope version, so the fixture bytes are deliberately the same for all
// four — that sameness IS the derived claim (derivation.md §2.5).
// ---------------------------------------------------------------------------

// mapleLifeGoldenBody: name="Chronicle" (9 ascii bytes), al=[1,2,3,4],
// gender=0, currentClass=100, sp=5, updateTime=123456789 (0x075BCD15).
var mapleLifeGoldenBody = []byte{
	0x09, 0x00, // sName length
	'C', 'h', 'r', 'o', 'n', 'i', 'c', 'l', 'e',
	0x01, 0x00, 0x00, 0x00, // al0 = 1
	0x02, 0x00, 0x00, 0x00, // al1 = 2
	0x03, 0x00, 0x00, 0x00, // al2 = 3
	0x04, 0x00, 0x00, 0x00, // al3 = 4
	0x00, 0x00, 0x00, 0x00, // nGender = 0
	0x64, 0x00, 0x00, 0x00, // nCurrentClass = 100
	0x05, 0x00, 0x00, 0x00, // nSP = 5
	0x15, 0xCD, 0x5B, 0x07, // update_time = 123456789
}

type imlFixtureVersion struct {
	name  string
	major uint16
	minor uint16
}

var imlFixtureVersions = []imlFixtureVersion{
	{"gms_v83", 83, 1},
	{"gms_v87", 87, 1},
	{"gms_v92", 92, 1},
	{"gms_v95", 95, 1},
}

func decodeIMLBytes(t *testing.T, name string, major uint16, minor uint16, body []byte) ItemUseMapleLife {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", major, minor)
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	out := *NewItemUseMapleLife(false)
	out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Errorf("%s: decoder left %d of %d bytes unconsumed", name, reader.Available(), len(body))
	}
	return out
}

func TestItemUseMapleLifeByteFixture(t *testing.T) {
	for _, v := range imlFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			out := decodeIMLBytes(t, v.name, v.major, v.minor, mapleLifeGoldenBody)
			if out.Name() != "Chronicle" {
				t.Errorf("name = %q, want %q", out.Name(), "Chronicle")
			}
			if out.AL0() != 1 || out.AL1() != 2 || out.AL2() != 3 || out.AL3() != 4 {
				t.Errorf("al = [%d %d %d %d], want [1 2 3 4]", out.AL0(), out.AL1(), out.AL2(), out.AL3())
			}
			if out.Gender() != 0 {
				t.Errorf("gender = %d, want 0", out.Gender())
			}
			if out.CurrentClass() != 100 {
				t.Errorf("currentClass = %d, want 100", out.CurrentClass())
			}
			if out.SP() != 5 {
				t.Errorf("sp = %d, want 5", out.SP())
			}
			if out.UpdateTime() != 123456789 {
				t.Errorf("updateTime = %d, want 123456789", out.UpdateTime())
			}
		})
	}
}

// TestItemUseMapleLifeFieldOrder decodes the golden bytes field-by-field
// (independently of TestItemUseMapleLifeByteFixture) so that a transposition
// of any two fields in item_use_maple_life.go's Encode/Decode would fail
// this test even if it happened to leave the OTHER test's assertions
// individually satisfiable by coincidence.
func TestItemUseMapleLifeFieldOrder(t *testing.T) {
	req := request.Request(mapleLifeGoldenBody)
	reader := request.NewRequestReader(&req, 0)

	name := reader.ReadAsciiString()
	al0 := reader.ReadInt32()
	al1 := reader.ReadInt32()
	al2 := reader.ReadInt32()
	al3 := reader.ReadInt32()
	gender := reader.ReadInt32()
	currentClass := reader.ReadInt32()
	sp := reader.ReadInt32()
	updateTime := reader.ReadUint32()

	if reader.Available() != 0 {
		t.Fatalf("manual decode left %d bytes unconsumed", reader.Available())
	}

	out := decodeIMLBytes(t, "gms_v95-order-check", 95, 1, mapleLifeGoldenBody)

	if out.Name() != name || out.AL0() != al0 || out.AL1() != al1 || out.AL2() != al2 || out.AL3() != al3 ||
		out.Gender() != gender || out.CurrentClass() != currentClass || out.SP() != sp || out.UpdateTime() != updateTime {
		t.Fatalf("codec field order does not match manual byte-order decode")
	}
}

func TestItemUseMapleLifeV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseMapleLife{name: "Chronicle", al0: 1, al1: 2, al2: 3, al3: 4, gender: 0, currentClass: 100, sp: 5, updateTime: 123456789}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(got, mapleLifeGoldenBody) {
		t.Fatalf("got % X, want % X", got, mapleLifeGoldenBody)
	}
}
