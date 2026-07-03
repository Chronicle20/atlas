package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// TestCharacterMovementByteOutput pins the CharacterMovement (MOVE_PLAYER) wire
// body. CUserRemote::OnMove is a thunk to CMovePath::OnMovePacket; the
// characterId(4) prefix is read by the pool dispatcher
// (CUserPool::OnUserRemotePacket) before dispatch, and the move-path body is the
// opaque CMovePath::OnMovePacket block:
//
//	v83 @0x9726ae, v87 @0x9f7647, v95 @0x948a80 — each a 1-line thunk to
//	CMovePath::OnMovePacket (verified at the named addresses).
//
// Wire = characterId(4 LE) + movePath blob. The move-path blob is an
// OPAQUE_LEDGER VERIFIED-EXCEPTION (docs/packets/audits/OPAQUE_LEDGER.md, "mob
// move-path" / "model.Movement shared path"): an element-loop register boundary
// with no per-field decompile line. Its bytes are the independently-audited
// model.Movement encoder (model/movement_test.go is the byte oracle). This test
// pins the non-opaque characterId prefix exactly and asserts the trailing blob
// equals model.Movement.Encode verbatim, plus a clean round-trip.
//
// normalTypesOptions returns a move-action "types" table whose index 0 is the
// NORMAL action, so model.Movement.Decode classifies an ElemType-0 element as a
// NormalElement (matching the encode shape) instead of a bare stub.
func normalTypesOptions() map[string]interface{} {
	return map[string]interface{}{
		"types": []interface{}{
			map[string]interface{}{"Name": "NORMAL", "Type": "NORMAL"},
		},
	}
}

// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v72 ida=0x87c1f8
// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v83 ida=0x9726ae
// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v84 ida=0x9b26cd
// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v87 ida=0x9f7647
// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v95 ida=0x948a80
// packet-audit:verify packet=character/clientbound/CharacterMovement version=jms_v185 ida=0xa443ee
//
// jms CUserRemote::OnMove@0xa443ee is a thunk to CMovePath::OnMovePacket@0x70c5dc,
// which calls CMovePath::Decode@0x70b3ce (the opaque move-path block) — byte-identical
// structure to v83/v87/v95: characterId(4 LE) prefix (read by the pool dispatcher) +
// the shared model.Movement blob. No wire delta on jms; the codec is version-agnostic.
func TestCharacterMovementByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range []struct {
		Name         string
		Region       string
		Major, Minor uint16
	}{
		{"GMS v72", "GMS", 72, 1},
		{"GMS v83", "GMS", 83, 1},
		{"GMS v84", "GMS", 84, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v95", "GMS", 95, 1},
		{"JMS v185", "JMS", 185, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.Major, v.Minor)
			// Move-action "types" table so Decode can classify element 0 as
			// NORMAL (the same options shape the live channel supplies).
			opts := normalTypesOptions()
			// Deterministic single-NORMAL-element path (no maps → byte-stable).
			mv := model.Movement{
				StartX: 100,
				StartY: 200,
				Elements: []model.MovementCodec{
					&model.NormalElement{Element: model.Element{
						ElemType: 0, X: 110, Y: 210, Vx: 5, Vy: -3, Fh: 1,
						BMoveAction: 7, TElapse: 50,
					}},
				},
			}
			in := NewCharacterMovement(0x01020304, mv)
			got := in.Encode(l, ctx)(opts)

			// characterId at [0:4], LE uint32.
			require.GreaterOrEqual(t, len(got), 4, "characterId prefix")
			require.Equal(t, []byte{0x04, 0x03, 0x02, 0x01}, got[0:4], "characterId LE uint32")

			// Trailing opaque move-path blob == model.Movement encoder output.
			wantBlob := mv.Encode(l, ctx)(opts)
			require.True(t, bytes.Equal(got[4:], wantBlob),
				"move-path blob must equal model.Movement encoder output\n got=% x\nwant=% x", got[4:], wantBlob)
			require.Equal(t, 4+len(wantBlob), len(got), "no trailing bytes after move-path")

			// Clean round-trip (no unconsumed bytes) proves the blob is self-sized.
			out := CharacterMovement{}
			pt.RoundTrip(t, ctx, in.Encode, out.Decode, opts)
			require.Equal(t, in.CharacterId(), out.CharacterId(), "characterId round-trip")
		})
	}
}

// TestCharacterMovementByteOutputV61 pins the very-legacy GMS v61 MOVE_PLAYER wire.
// IDA-verified: CUserRemote::OnMove @0x7bdd6b (GMS_v61.1_U_DEVM.exe, port 13338 —
// registry's dispatcher note-address 0x7bd8eb is the pool switch, not the handler) is a
// thunk to CMovePath::OnMovePacket @0x5e3770 — byte-identical structure to v72/v83:
// characterId(4 LE) prefix (read by the pool dispatcher) + the shared model.Movement
// opaque blob. model.Movement has no version gate, so the blob is byte-identical to v72
// (OPAQUE_LEDGER VERIFIED-EXCEPTION; model/movement_test.go is the byte oracle).
// packet-audit:verify packet=character/clientbound/CharacterMovement version=gms_v61 ida=0x7bdd6b
func TestCharacterMovementByteOutputV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	opts := normalTypesOptions()
	mv := model.Movement{
		StartX: 100,
		StartY: 200,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{
				ElemType: 0, X: 110, Y: 210, Vx: 5, Vy: -3, Fh: 1,
				BMoveAction: 7, TElapse: 50,
			}},
		},
	}
	in := NewCharacterMovement(0x01020304, mv)
	got := in.Encode(l, ctx)(opts)

	require.GreaterOrEqual(t, len(got), 4, "characterId prefix")
	require.Equal(t, []byte{0x04, 0x03, 0x02, 0x01}, got[0:4], "characterId LE uint32")

	wantBlob := mv.Encode(l, ctx)(opts)
	require.True(t, bytes.Equal(got[4:], wantBlob),
		"move-path blob must equal model.Movement encoder output\n got=% x\nwant=% x", got[4:], wantBlob)
	require.Equal(t, 4+len(wantBlob), len(got), "no trailing bytes after move-path")

	out := CharacterMovement{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, opts)
	require.Equal(t, in.CharacterId(), out.CharacterId(), "characterId round-trip")
}
