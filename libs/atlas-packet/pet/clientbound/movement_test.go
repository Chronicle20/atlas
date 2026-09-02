package clientbound

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// movingElement is a single NORMAL element (index 0 of test.MovementTypesV95,
// which is NORMAL/NORMAL for every client version — see
// character/clientbound/movement_test.go's normalTypesOptions), so the round
// trip below exercises model.Movement's element loop under the real v95
// `types` table instead of an empty, element-less Movement.
func movingElement() model.Movement {
	return model.Movement{
		StartX: 100,
		StartY: 200,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{
				ElemType: 0, X: 110, Y: 210, Vx: 5, Vy: -3, Fh: 1,
				BMoveAction: 7, TElapse: 50,
			}},
		},
	}
}

// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v83 ida=0x70474d
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v87 ida=0x74842a
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v95 ida=0x69fb60
// packet-audit:verify packet=pet/clientbound/PetMovement version=jms_v185 ida=0x76a534
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v84 ida=0x720e70
func TestPetMovementRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mv := movingElement()
			input := NewPetMovement(2001, 0, mv)
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// MovementRoundTrip rather than RoundTrip because clientbound movement encode is not
			// the inverse of decode on GMS v87 — see its doc comment.
			test.MovementRoundTrip(t, ctx, input.Encode, input.Decode, test.MovementTypesV95())
			// MovementRoundTrip skips the encode/decode identity assertion on
			// GMS v87 (see its doc comment), so this blob assertion is what keeps
			// the v87 subtest — and its packet-audit:verify marker — honest.
			assertMovePathBlob(t, ctx, input.Encode, mv, test.MovementTypesV95())
		})
	}
}

// TestPetMovementHighestIndexResolves pins index 36 — the highest index in
// test.MovementTypesV95() — to its real resolved shape (NORMAL) rather than
// the NOT_FOUND/DEFAULT fallback model.resolveMovementPathAttr returns for an
// attribute byte outside the table. This is the coupling task-146 Task 11
// added as a substitute for a `types` CI gate: an off-by-one or truncated
// array desyncs the array from the client's real 37-entry table (index 36 =
// NORMAL, v95-option-tables.md), so decode falls back to a bare
// model.Element (no NORMAL fields, no XOffset/YOffset), and this test's
// round-trip and type assertion both fail.
func TestPetMovementHighestIndexResolves(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	mv := model.Movement{
		StartX: 100,
		StartY: 200,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{
				ElemType: 36, X: 110, Y: 210, Vx: 5, Vy: -3, Fh: 1,
				XOffset: 2, YOffset: -2, BMoveAction: 7, TElapse: 50,
			}},
		},
	}
	input := NewPetMovement(2001, 0, mv)

	var out Movement
	test.RoundTrip(t, ctx, input.Encode, out.Decode, test.MovementTypesV95())

	require.Len(t, out.movement.Elements, 1)
	elem, ok := out.movement.Elements[0].(*model.NormalElement)
	require.True(t, ok, "index 36 must resolve to NORMAL (NormalElement), not the DEFAULT fallback")
	require.Equal(t, int16(110), elem.X)
	require.Equal(t, int16(210), elem.Y)
	require.Equal(t, int16(5), elem.Vx)
	require.Equal(t, int16(-3), elem.Vy)
	require.Equal(t, int16(1), elem.Fh)
	require.Equal(t, int16(2), elem.XOffset)
	require.Equal(t, int16(-2), elem.YOffset)
}

// v79 MOVE_PET (cb op 0xAA=170) read order, verified GMS_v79_1_DEVM.exe (port
// 13340): CUserPool::OnUserCommonPacket@0x8c8c79 Decode4(ownerId)@0x8c8c84 →
// CUser::OnPetPacket@0x892474 Decode1(slot)@0x8924b6 → CPet::OnMove@0x690ecb →
// CMovePath::OnMovePacket (opaque movement). Wire = ownerId(4) + slot(1) +
// movement. Codec is version-unconditional; empty model.Movement encodes to
// StartX(2)+StartY(2)+count(1) = 5 zero bytes. Layout byte-identical to v83.
// TestPetMovementBytesV72 pins the v72 wire = v79 (no version gate on the codec).
// IDA GMS_v72.1_U_DEVM.exe @port 13339: CPet::OnMove@0x66c083 forwards the
// CInPacket to CMovePath::OnMovePacket@0x635bc2 (raw movement blob). ownerId +
// slot are read upstream by CUser::OnPetPacket before the leaf dispatch.
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v72 ida=0x66c083
func TestPetMovementBytesV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	got := NewPetMovement(0x01020304, 0x05, model.Movement{}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,       // slot (upstream)
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v79 ida=0x690ecb
func TestPetMovementBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	got := NewPetMovement(0x01020304, 0x05, model.Movement{}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId Decode4@0x8c8c84 (LE)
		0x05,       // slot Decode1@0x8924b6
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}

// assertMovePathBlob pins the move-path blob this packet emits to the
// independently-audited model.Movement encoder output for the SAME tenant, and
// requires it to be the packet's suffix.
//
// It is the v87 safety net for test.MovementRoundTrip, which cannot assert
// encode/decode identity on that version: GMS v87 reads the per-element
// XOffset/YOffset pair but is never sent it back (CMovePath::Encode @0x6c70fe
// vs CMovePath::Decode @0x6c6e86). Without this the GMS v87 subtest would run
// zero assertions while still carrying a packet-audit:verify marker. The blob's
// per-version width is pinned in libs/atlas-packet/model/version_bounds_test.go.
func assertMovePathBlob(t *testing.T, ctx context.Context, encode func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, mv model.Movement, opts map[string]interface{}) {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	got := encode(l, ctx)(opts)
	want := mv.Encode(l, ctx)(opts)
	require.True(t, bytes.HasSuffix(got, want),
		"move-path blob must equal model.Movement encoder output\n got=% x\nwant=% x", got, want)
}

// TestPetMovementV87TemplateOptionsCarryFhFallStart drives the writer with the
// OPTIONS THE SEED TEMPLATE ACTUALLY REGISTERS IT WITH, and pins the one field
// whose presence depends on them: NormalElement.Encode writes FhFallStart only
// when the fragment's attr resolves through options.types to the reserved name
// FALL_DOWN (index 15 in template_gms_87_1.json's 25-entry table). With no
// table the check never fires and fhFallStart is silently dropped from the
// outbound fragment — no error, no log line, just a short fragment on the wire,
// which is what template_gms_87_1.json shipped before this test existed.
func TestPetMovementV87TemplateOptionsCarryFhFallStart(t *testing.T) {
	ctx := test.CreateContext("GMS", 87, 1)
	mv := model.Movement{
		StartX: 100,
		StartY: 200,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{
				ElemType: 15, X: 110, Y: 210, Vx: 5, Vy: -3, Fh: 1,
				FhFallStart: 9, BMoveAction: 7, TElapse: 50,
			}},
		},
	}
	input := NewPetMovement(2001, 0, mv)

	options := test.TemplateWriterOptions(t, "template_gms_87_1.json", PetMovementWriter)
	got := test.Encode(t, ctx, input.Encode, options)

	want := []byte{
		0xD1, 0x07, 0x00, 0x00, // ownerId 2001
		0x00,       // slot 0
		0x64, 0x00, // StartX 100
		0xC8, 0x00, // StartY 200
		0x01,       // one fragment
		0x0F,       // attr 15 == FALL_DOWN, resolves to NORMAL
		0x6E, 0x00, // X 110
		0xD2, 0x00, // Y 210
		0x05, 0x00, // Vx 5
		0xFD, 0xFF, // Vy -3
		0x01, 0x00, // Fh 1
		0x09, 0x00, // FhFallStart 9 — present only because attr 15 is named FALL_DOWN
		0x07,       // bMoveAction 7 (no XOffset/YOffset: v87 never reads them back)
		0x32, 0x00, // tElapse 50
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 pet movement with template options = % X, want % X", got, want)
	}
}
