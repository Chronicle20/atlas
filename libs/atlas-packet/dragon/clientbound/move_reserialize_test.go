package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// dragonMoveHeader is the fixed ownerCharacterId prefix (4242) the
// reserialization tests below skip past to get at the move-path blob.
var dragonMoveHeader = []byte{0x92, 0x10, 0x00, 0x00}

// TestDragonMoveV87DropsElementOffsets proves the v87 rebroadcast no longer
// echoes the per-element XOffset/YOffset pair the v87 client sent.
//
// The v87 client WRITES the pair (CMovePath::Encode @0x6c70fe, mov ax,[edi+14h]
// / [edi+16h] at 6c720a/6c7218) and NEVER READS it (CMovePath::Decode @0x6c6e86
// goes from fh straight to the bMoveAction/tElapse tail). CDragon::OnMove
// @0x520c71 hands the body straight to CMovePath::OnMovePacket, so echoing the
// capture verbatim desynced every observing client's fragment loop.
//
// Expected is the hand-built v83-shaped fragment (the layout v87's Decode
// actually reads), not the encoder's own output. The trailer must survive
// byte-for-byte: Atlas does not model it and must not drop it.
func TestDragonMoveV87DropsElementOffsets(t *testing.T) {
	captured := test.MovePathBlob(false, true)
	rebroadcast := test.MovePathBlob(false, false)

	got := test.Encode(t, test.CreateContext("GMS", 87, 1),
		NewDragonMove(4242, captured).Encode, test.MovementTypesV95())

	want := append(append([]byte{}, dragonMoveHeader...), rebroadcast...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 dragon move bytes = % X, want % X", got, want)
	}
	if bytes.Contains(got, captured) {
		t.Errorf("v87 dragon move still carries the captured blob verbatim: % X", got)
	}
	if !bytes.HasSuffix(got, test.MovePathTrailer) {
		t.Errorf("v87 dragon move dropped the move-path trailer: % X", got)
	}
}

// TestDragonMoveSymmetricVersionsAreByteUnchanged pins every version whose
// CMovePath reads back exactly what it writes: re-serializing must be a no-op
// there, so the emitted blob equals the captured one byte-for-byte.
//
// Per-version layout evidence (CMovePath::Encode / ::Decode):
//
//	GMS v83  @0x68a563 / @0x68a33c — no start velocity, no element offsets
//	GMS v92  @0x65a260 / @0x65ad60 — start velocity, element offsets
//	GMS v95  @0x666e20 / @0x667920 — start velocity, element offsets
//	JMS v185 @0x70b6c4 / @0x70b3ce — no start velocity, element offsets
func TestDragonMoveSymmetricVersionsAreByteUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name           string
		region         string
		major          uint16
		startVelocity  bool
		elementOffsets bool
	}{
		{"gms_v83", "GMS", 83, false, false},
		{"gms_v92", "GMS", 92, true, true},
		{"gms_v95", "GMS", 95, true, true},
		{"jms_v185", "JMS", 185, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			captured := test.MovePathBlob(tc.startVelocity, tc.elementOffsets)
			got := test.Encode(t, test.CreateContext(tc.region, tc.major, 1),
				NewDragonMove(4242, captured).Encode, test.MovementTypesV95())

			want := append(append([]byte{}, dragonMoveHeader...), captured...)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s dragon move bytes = % X, want the captured blob unchanged % X", tc.name, got, want)
			}
		})
	}
}

// TestDragonMoveUnparseableBlobIsUnchanged pins the fallback: when the fragment
// attrs are not in the tenant's types table their widths are unknown, so
// re-serializing would truncate. The capture is rebroadcast as-is instead.
func TestDragonMoveUnparseableBlobIsUnchanged(t *testing.T) {
	captured := test.MovePathBlob(false, true)
	got := test.Encode(t, test.CreateContext("GMS", 87, 1),
		NewDragonMove(4242, captured).Encode, nil)

	want := append(append([]byte{}, dragonMoveHeader...), captured...)
	if !bytes.Equal(got, want) {
		t.Fatalf("dragon move with no types table = % X, want the captured blob unchanged % X", got, want)
	}
}
