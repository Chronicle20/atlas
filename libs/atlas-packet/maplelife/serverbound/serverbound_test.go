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
// CheckName — CUICharacterSaleDlg::SendCheckDuplicateIDPacket, the Maple
// Life duplicate-name probe.
//
// FIVE in-scope cells: gms_v83, gms_v84, gms_v87, gms_v92, gms_v95 —
// CUICharacterSaleDlg exists on gms_v84 too, at its own opcode 263 (0x107);
// an earlier pass's VERSION-ABSENT finding was wrong and has been retracted
// (derivation.md §2.0-CORRECTION, supersedes §6.1's four-cell framing; see
// also docs/tasks/task-246-maple-life-character-creation/bug-maple-life-v84-registration.md).
// The byte-fixture list below now covers all five cells; gms_v84's wire
// framing is identical to the rest (a single EncodeStr, no per-version
// field-shape difference).
//
// The `ida=` address on each marker is that version's SENDER, decompiled
// this pass: gms_v83 @0x7d75ab, gms_v84 @0x7fd86a, gms_v87 @0x82e04d,
// gms_v92 @0x756250 (unnamed sub_756250 renamed to the mangled
// CUICharacterSaleDlg::SendCheckDuplicateIDPacket symbol this pass),
// gms_v95 @0x777d20. Opcodes per version: v83=256, v84=263, v87=270,
// v92=301, v95=311 (derivation.md §6.1; gms_v84 per
// docs/tasks/task-246-maple-life-character-creation/bug-maple-life-v84-registration.md,
// which retracted an earlier VERSION-ABSENT finding for v84) — each
// version's own dedicated opcode, no CHECK_CHAR_NAME(21) collision on any
// of them.
// ---------------------------------------------------------------------------

// packet-audit:verify packet=maplelife/serverbound/MaplelifeCheckName version=gms_v83 ida=0x7d75ab
// packet-audit:verify packet=maplelife/serverbound/MaplelifeCheckName version=gms_v84 ida=0x7fd86a
// packet-audit:verify packet=maplelife/serverbound/MaplelifeCheckName version=gms_v87 ida=0x82e04d
// packet-audit:verify packet=maplelife/serverbound/MaplelifeCheckName version=gms_v92 ida=0x756250
// packet-audit:verify packet=maplelife/serverbound/MaplelifeCheckName version=gms_v95 ida=0x777d20
func TestMapleLifeCheckNameRoundTrip(t *testing.T) {
	inScope := []pt.TenantVariant{
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v84", Region: "GMS", MajorVersion: 84, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "GMS v92", Region: "GMS", MajorVersion: 92, MinorVersion: 1},
		{Name: "GMS v95", Region: "GMS", MajorVersion: 95, MinorVersion: 1},
	}
	for _, v := range inScope {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCheckName("Chronicle")
			output := CheckName{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.Name() != input.Name() {
				t.Errorf("name = %q, want %q", output.Name(), input.Name())
			}
		})
	}
}

func TestMapleLifeCheckNameOperation(t *testing.T) {
	m := CheckName{}
	if m.Operation() != MapleLifeCheckNameHandle {
		t.Errorf("Operation() = %q, want %q", m.Operation(), MapleLifeCheckNameHandle)
	}
	if MapleLifeCheckNameHandle != "MapleLifeCheckNameHandle" {
		t.Errorf("handle name = %q", MapleLifeCheckNameHandle)
	}
}

// ---------------------------------------------------------------------------
// Byte fixtures.
//
// Framing: WriteAsciiString is a uint16 LE length prefix + raw bytes.
// Field order: derivation.md §6 (a single EncodeStr sCharName, no other
// fields) — IDENTICAL SHAPE on every in-scope version, so the fixture is
// deliberately the same bytes for all five.
// ---------------------------------------------------------------------------

// mapleLifeCheckNameGoldenBody: sCharName = "Chronicle" (9 ascii bytes).
var mapleLifeCheckNameGoldenBody = []byte{
	0x09, 0x00,
	'C', 'h', 'r', 'o', 'n', 'i', 'c', 'l', 'e',
}

type mlcnFixtureVersion struct {
	name  string
	major uint16
	minor uint16
}

var mlcnFixtureVersions = []mlcnFixtureVersion{
	{"gms_v83", 83, 1},
	{"gms_v84", 84, 1},
	{"gms_v87", 87, 1},
	{"gms_v92", 92, 1},
	{"gms_v95", 95, 1},
}

func TestMapleLifeCheckNameByteFixture(t *testing.T) {
	for _, v := range mlcnFixtureVersions {
		t.Run(v.name, func(t *testing.T) {
			l, _ := testlog.NewNullLogger()
			ctx := pt.CreateContext("GMS", v.major, v.minor)
			req := request.Request(mapleLifeCheckNameGoldenBody)
			reader := request.NewRequestReader(&req, 0)
			var out CheckName
			out.Decode(logrus.FieldLogger(l), ctx)(&reader, nil)
			if reader.Available() != 0 {
				t.Errorf("%s: decoder left %d of %d bytes unconsumed", v.name, reader.Available(), len(mapleLifeCheckNameGoldenBody))
			}
			if out.Name() != "Chronicle" {
				t.Errorf("name = %q, want %q", out.Name(), "Chronicle")
			}
		})
	}
}

func TestMapleLifeCheckNameV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewCheckName("Chronicle")
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(got, mapleLifeCheckNameGoldenBody) {
		t.Fatalf("got % X, want % X", got, mapleLifeCheckNameGoldenBody)
	}
}
