package test

import "encoding/binary"

// MovePathBlob assembles, by hand, the CMovePath blob a client sends for a
// single NORMAL (attr 0) movement fragment, followed by a short opaque trailer
// standing in for the keypad-input run and the m_rcMove bounding box that
// CMovePath::Encode always writes after the fragment array.
//
// The two layout choices are passed in explicitly rather than derived from a
// tenant, so a test states the layout it believes a given client version uses
// and fails if Atlas disagrees — the point being to assert against the
// pre-existing wire layout, not against the encoder's own output:
//
//   - startVelocity: the header carries startVx/startVy after startX/startY.
//     GMS v92 @0x65a260 and v95 @0x666e20 write it; GMS v83 @0x68a563,
//     v87 @0x6c70fe and JMS v185 @0x70b6c4 write startX/startY/count only.
//   - elementOffsets: the fragment carries the XOffset/YOffset pair between fh
//     and the bMoveAction/tElapse tail. GMS v87 @0x6c70fe, v92, v95 and JMS
//     v185 write it; GMS v83/v84 have no such field.
//
// Byte values are arbitrary but distinct so a misplaced field is visible in a
// failure diff.
func MovePathBlob(startVelocity bool, elementOffsets bool) []byte {
	b := make([]byte, 0, 32)
	b = appendInt16(b, 100) // startX
	b = appendInt16(b, 200) // startY
	if startVelocity {
		b = appendInt16(b, 3)  // startVx
		b = appendInt16(b, -4) // startVy
	}
	b = append(b, 0x01)     // fragment count
	b = append(b, 0x00)     // attr 0 == NORMAL
	b = appendInt16(b, 110) // x
	b = appendInt16(b, 210) // y
	b = appendInt16(b, 1)   // vx
	b = appendInt16(b, -2)  // vy
	b = appendInt16(b, 7)   // fh
	if elementOffsets {
		b = appendInt16(b, 6) // xOffset
		b = appendInt16(b, 7) // yOffset
	}
	b = append(b, 0x03)    // bMoveAction
	b = appendInt16(b, 17) // tElapse
	return append(b, MovePathTrailer...)
}

// MovePathTrailer is the opaque tail of a MovePathBlob: everything Atlas's
// move-path codec does not model. A rebroadcast must carry it through unchanged.
var MovePathTrailer = []byte{0x00, 0xAA, 0xBB, 0xCC, 0xDD}

func appendInt16(b []byte, v int16) []byte {
	return binary.LittleEndian.AppendUint16(b, uint16(v))
}
