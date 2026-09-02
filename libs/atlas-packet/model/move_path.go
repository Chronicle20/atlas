package model

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// movePathTrailerMinBytes is the narrowest trailer CMovePath::Encode can write
// after the fragment array. That encoder ALWAYS emits, in order, Encode1
// keypadLen, the keypad run (which may be empty), then four Encode2 for
// m_rcMove (minX/minY/maxX/maxY) — so never fewer than 1 + 0 + 8 = 9 bytes.
// (CMovePath::Encode @0x68a563; docs/tasks/task-088-player-summons/
// summon-wire-truth.md, "Move send"; libs/atlas-packet/summon/serverbound/move.go.)
//
// It is the over-read detector this codec otherwise lacks: request.Reader
// returns 0 for a read past the end WITHOUT advancing, so a fragment truncated
// mid-field decodes into a fabricated one and leaves the reader stalled short of
// the blob's end. A decode that ends with fewer than nine bytes left therefore
// did not land on the real end of the fragment array, whatever it produced, and
// re-encoding from that state would broadcast bytes the client never sent.
const movePathTrailerMinBytes = 9

// reserializeLogger is the logger Movement.Decode is given on the re-serialize
// (writer) path, and it discards everything.
//
// Movement.Decode reports an out-of-table fragment code with an Errorf carrying
// a hex dump of the frame (logUnconfiguredMovementCode). That is load-bearing on
// the INBOUND handler path — it is what made the GMS v87 misalignment
// diagnosable — but this path runs inside the encoder, once per RECEIVING
// session, at movement-packet rate; the same summon would log once per element
// per observer. And there is nothing to act on: an attr this path cannot resolve
// takes the fallback below and the capture ships verbatim.
//
// It is deliberately a package-level value rather than a parameter, so no future
// caller can hand ReserializeMovePath a noisy logger and get the flood back.
var reserializeLogger = newDiscardLogger()

func newDiscardLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetLevel(logrus.PanicLevel)
	return l
}

// ReserializeMovePath re-encodes a client-captured CMovePath blob so that the
// bytes Atlas rebroadcasts follow the OUTBOUND element layout for the tenant,
// instead of echoing back the INBOUND layout the client sent.
//
// It exists for the packets that capture a move-path as an opaque []byte and
// rebroadcast it (summon and dragon movement). Those never went through
// Movement's codec, so on GMS v87 they still carried the per-element
// XOffset/YOffset pair that v87's CMovePath::Decode @0x6c6e86 never reads —
// the observing client then reads xOffset's low byte as bMoveAction and yOffset
// as tElapse and the whole element loop desyncs. See the evidence table above
// movementElementOffsetsInbound/movementElementOffsetsOutbound in movement.go.
//
// Running the blob through Movement.Decode + Movement.Encode inherits that
// inbound/outbound split for free: on every version where the two gates agree
// (GMS v83/v84 no, GMS v92+/JMS yes) the re-encoded prefix is byte-identical to
// what came in, and only GMS v87 loses the four bytes it must not be sent.
//
// The trailing bytes CMovePath::Encode writes after the element array (the
// keypad-input run and the m_rcMove bounding box) are NOT parsed by
// Movement.Decode and are NOT reconstructed here — they are carried through
// verbatim from the input blob. Whether the receiving client reads that trailer
// depends on CMovePath::Decode's bPassive argument, which is not established
// for every version in this matrix; copying the bytes through unchanged keeps
// the existing behaviour on every version rather than depending on the answer.
//
// The blob is returned unchanged whenever it cannot be parsed with confidence:
//   - it is too short to hold a header,
//   - the decode consumed nothing, or left fewer than
//     movePathTrailerMinBytes behind (a real client blob always has a trailer
//     at least that wide, so a shorter remainder means the fragment array did
//     not end where the decode thinks it did), or
//   - any element fell back to the bare Element shape, meaning its attr is not
//     in the tenant's "types" table and its true width is unknown. Re-encoding
//     from that state would truncate the fragment.
//
// options must be the tenant's own movement options ("types"), the same table
// Movement.Decode consumes; the blob is produced and consumed by clients of a
// single tenant, so no cross-version translation is involved.
func ReserializeMovePath(l logrus.FieldLogger, ctx context.Context) func(raw []byte, options map[string]interface{}) []byte {
	return func(raw []byte, options map[string]interface{}) []byte {
		// x, y and the element count are the smallest header any version writes.
		if len(raw) < 5 {
			return raw
		}

		req := request.Request(raw)
		r := request.NewRequestReader(&req, 0)
		var m Movement
		// reserializeLogger, not l: see its declaration. An unresolvable attr
		// here is handled by the fallback below, not by an operator.
		m.Decode(reserializeLogger, ctx)(&r, options)

		consumed := r.Position()
		if consumed <= 0 || len(raw)-consumed < movePathTrailerMinBytes {
			return raw
		}
		if len(m.Elements) == 0 {
			return raw
		}
		for _, e := range m.Elements {
			if _, ok := e.(*Element); ok {
				return raw
			}
		}

		body := m.Encode(l, ctx)(options)
		out := make([]byte, 0, len(body)+len(raw)-consumed)
		out = append(out, body...)
		out = append(out, raw[consumed:]...)
		return out
	}
}
