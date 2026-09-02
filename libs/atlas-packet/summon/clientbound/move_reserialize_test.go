package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// header is the fixed cid + oid prefix the reserialization tests below skip past
// to get at the move-path blob.
var summonMoveHeader = []byte{0x2A, 0x00, 0x00, 0x00, 0x41, 0x42, 0x0F, 0x00}

// TestSummonMoveV87DropsElementOffsets proves the v87 rebroadcast no longer
// echoes the per-element XOffset/YOffset pair the v87 client sent.
//
// The v87 client WRITES the pair (CMovePath::Encode @0x6c70fe, mov ax,[edi+14h]
// / [edi+16h] at 6c720a/6c7218) and NEVER READS it (CMovePath::Decode @0x6c6e86
// goes from fh straight to the bMoveAction/tElapse tail). Rebroadcasting the
// capture verbatim therefore made every observing client read xOffset's low byte
// as bMoveAction and yOffset as tElapse, desyncing the whole fragment loop —
// summons teleported for everyone except their owner, who renders locally and is
// never sent this packet.
//
// Expected is the hand-built v83-shaped fragment (the layout v87's Decode
// actually reads), not the encoder's own output. The trailer must survive
// byte-for-byte: Atlas does not model it and must not drop it.
func TestSummonMoveV87DropsElementOffsets(t *testing.T) {
	captured := test.MovePathBlob(false, true)
	rebroadcast := test.MovePathBlob(false, false)

	got := test.Encode(t, test.CreateContext("GMS", 87, 1),
		NewSummonMove(42, 1000001, captured).Encode, test.MovementTypesV95())

	want := append(append([]byte{}, summonMoveHeader...), rebroadcast...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 summon move bytes = % X, want % X", got, want)
	}
	if bytes.Contains(got, captured) {
		t.Errorf("v87 summon move still carries the captured blob verbatim: % X", got)
	}
	if !bytes.HasSuffix(got, test.MovePathTrailer) {
		t.Errorf("v87 summon move dropped the move-path trailer: % X", got)
	}
}

// TestSummonMoveSymmetricVersionsAreByteUnchanged pins every version whose
// CMovePath reads back exactly what it writes: re-serializing must be a no-op
// there, so the emitted blob equals the captured one byte-for-byte.
//
// Per-version layout evidence (CMovePath::Encode / ::Decode):
//
//	GMS v83  @0x68a563 / @0x68a33c — no start velocity, no element offsets
//	GMS v92  @0x65a260 / @0x65ad60 — start velocity, element offsets
//	GMS v95  @0x666e20 / @0x667920 — start velocity, element offsets
//	JMS v185 @0x70b6c4 / @0x70b3ce — no start velocity, element offsets
func TestSummonMoveSymmetricVersionsAreByteUnchanged(t *testing.T) {
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
				NewSummonMove(42, 1000001, captured).Encode, test.MovementTypesV95())

			want := append(append([]byte{}, summonMoveHeader...), captured...)
			if !bytes.Equal(got, want) {
				t.Fatalf("%s summon move bytes = % X, want the captured blob unchanged % X", tc.name, got, want)
			}
		})
	}
}

// TestSummonMoveUnparseableBlobIsUnchanged pins the fallback: when the fragment
// attrs are not in the tenant's types table their widths are unknown, so
// re-serializing would truncate. The capture is rebroadcast as-is instead.
func TestSummonMoveUnparseableBlobIsUnchanged(t *testing.T) {
	captured := test.MovePathBlob(false, true)
	got := test.Encode(t, test.CreateContext("GMS", 87, 1),
		NewSummonMove(42, 1000001, captured).Encode, nil)

	want := append(append([]byte{}, summonMoveHeader...), captured...)
	if !bytes.Equal(got, want) {
		t.Fatalf("summon move with no types table = % X, want the captured blob unchanged % X", got, want)
	}
}

// TestSummonMoveV87TemplateOptionsDropElementOffsets is the same assertion as
// TestSummonMoveV87DropsElementOffsets, driven from the OPTIONS THE SEED
// TEMPLATE ACTUALLY REGISTERS THE WRITER WITH instead of a hand-built table.
//
// That distinction is the whole point. The fix in bd3a09003 was inert on a live
// GMS 87.1 tenant for exactly one reason: template_gms_87_1.json's SummonMove
// writer entry carried no options.types, so ReserializeMovePath could not
// classify the fragment, took its unparseable-blob fallback, and reshipped the
// capture verbatim — while the hand-fed test above went on passing. This test
// fails if the table is ever dropped from the template again.
func TestSummonMoveV87TemplateOptionsDropElementOffsets(t *testing.T) {
	captured := test.MovePathBlob(false, true)
	rebroadcast := test.MovePathBlob(false, false)

	options := test.TemplateWriterOptions(t, "template_gms_87_1.json", SummonMoveWriter)
	got := test.Encode(t, test.CreateContext("GMS", 87, 1),
		NewSummonMove(42, 1000001, captured).Encode, options)

	want := append(append([]byte{}, summonMoveHeader...), rebroadcast...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 summon move with template options = % X, want % X", got, want)
	}
	if bytes.Contains(got, captured) {
		t.Errorf("v87 summon move with template options still carries the captured blob verbatim: % X", got)
	}
	if !bytes.HasSuffix(got, test.MovePathTrailer) {
		t.Errorf("v87 summon move with template options dropped the move-path trailer: % X", got)
	}
}
