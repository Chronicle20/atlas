package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

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
//   - the decode consumed nothing, or consumed the whole blob leaving no
//     trailer (a real client blob always has one), or
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
		m.Decode(l, ctx)(&r, options)

		consumed := r.Position()
		if consumed <= 0 || consumed >= len(raw) {
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
