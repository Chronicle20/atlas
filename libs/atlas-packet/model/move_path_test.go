package model

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// truncatedMovePathBlob builds a client-controlled blob that used to slip past
// ReserializeMovePath's "cannot parse with confidence" guard: a header claiming
// TWO fragments, one complete v87 NORMAL fragment, and a second fragment cut
// off inside its trailing tElapse — with no move-path trailer at all.
//
// request.Reader returns 0 for a read past the end WITHOUT advancing, so
// fragment 2's tElapse decodes as a FABRICATED zero and the reader stalls at
// offset 39 of 40. The old guard only rejected consumed >= len(raw), so it did
// not fire: Atlas re-encoded the invented fragment and broadcast it to every
// observing session, with the one unread byte tacked on as a "trailer".
func truncatedMovePathBlob() []byte {
	b := []byte{
		100, 0, // startX
		200, 0, // startY
		0x02, // fragment count — claims two

		0x00,   // fragment 1: attr 0 == NORMAL
		110, 0, // x
		210, 0, // y
		1, 0, // vx
		254, 255, // vy
		7, 0, // fh
		6, 0, // xOffset
		7, 0, // yOffset
		0x03,  // bMoveAction
		17, 0, // tElapse

		0x00,   // fragment 2: attr 0 == NORMAL
		111, 0, // x
		211, 0, // y
		1, 0, // vx
		254, 255, // vy
		7, 0, // fh
		6, 0, // xOffset
		7, 0, // yOffset
		0x03, // bMoveAction
		18,   // tElapse, cut off after one byte — the blob ends here
	}
	return b
}

// TestReserializeMovePathTruncatedFragmentIsUnchanged pins the documented
// contract: a blob that cannot be parsed with confidence goes out verbatim.
//
// The floor is CMovePath::Encode's own trailer, which is never shorter than
// nine bytes (Encode1 keypadLen, the keypad run, four Encode2 for m_rcMove), so
// a parse that ends with fewer than nine bytes left did not land on the real end
// of the fragment array.
func TestReserializeMovePathTruncatedFragmentIsUnchanged(t *testing.T) {
	raw := truncatedMovePathBlob()
	if len(raw) != 40 {
		t.Fatalf("fixture is %d bytes, want the 40-byte truncated shape", len(raw))
	}

	l, _ := testlog.NewNullLogger()
	got := ReserializeMovePath(l, test.CreateContext("GMS", 87, 1))(raw, test.MovementTypesV95())

	if !bytes.Equal(got, raw) {
		t.Fatalf("truncated blob was re-serialized to % X, want the capture unchanged % X", got, raw)
	}
}

// TestReserializeMovePathWellFormedBlobStillReserializes guards the floor above
// from being set so high that it swallows a real capture: the v87 blob the
// summon and dragon writers exist to fix must still lose its XOffset/YOffset
// pair.
func TestReserializeMovePathWellFormedBlobStillReserializes(t *testing.T) {
	captured := test.MovePathBlob(false, true)

	l, _ := testlog.NewNullLogger()
	got := ReserializeMovePath(l, test.CreateContext("GMS", 87, 1))(captured, test.MovementTypesV95())

	want := test.MovePathBlob(false, false)
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 re-serialized blob = % X, want % X", got, want)
	}
}

// TestReserializeMovePathDoesNotLogUnconfiguredCodes pins the writer path's
// silence. Movement.Decode's unconfigured-code diagnostic is an Errorf carrying
// a hex dump of the frame, and session.Announce runs the encoder once per
// RECEIVING session — so on this path one summon with an out-of-table attr
// would emit that error once per element per observer, at movement-packet rate.
// The unresolvable attr is already handled here by shipping the blob verbatim,
// so there is nothing for an operator to act on. The diagnostic stays loud on
// the inbound handler path, which is what made the v87 flood diagnosable.
func TestReserializeMovePathDoesNotLogUnconfiguredCodes(t *testing.T) {
	l, hook := testlog.NewNullLogger()

	captured := test.MovePathBlob(false, true)
	got := ReserializeMovePath(l, test.CreateContext("GMS", 87, 1))(captured, nil)

	if !bytes.Equal(got, captured) {
		t.Fatalf("blob with no types table = % X, want the capture unchanged % X", got, captured)
	}
	if entries := hook.AllEntries(); len(entries) != 0 {
		t.Fatalf("re-serialize logged %d entries on the writer path, want none: %v", len(entries), entries[0].Message)
	}
}
