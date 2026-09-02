package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetMovementHandle = "PetMovementHandle"

// petMoveKeyPadTail reports whether the serverbound MOVE_PET movement blob
// ends with the keypad history + path bounding rectangle that
// CMovePath::Encode appends after the element loop (see moveKeyPadTail in
// character/serverbound/move.go for the full field derivation:
// count-of-entries byte, ceil(count/2) packed nibble bytes, then left/top/
// right/bottom int16).
//
// Decompile-confirmed for jms v185, NOT live-confirmed:
// CVecCtrlPet::EndUpdateActive @0xaa25ab writes EncodeBuffer(petId, 8)
// @0xaa25fc then CMovePath::Flush @0xaa2609 — the same encoder that
// unconditionally appends this tail (see character/serverbound/move.go's
// moveKeyPadTail). Header matches Atlas exactly; scope is tail-only. See
// docs/tasks/fix-jms185-attack-decode/sibling-movement-ops-findings.md §3.
//
// Gated to JMS because that is the only client this sender was read on. Do
// not extend to GMS without reading each GMS version's
// CVecCtrlPet::EndUpdateActive directly.
func petMoveKeyPadTail(t tenant.Model) bool {
	return t.IsRegion("JMS")
}

// packet-audit:fname CVecCtrlPet::EndUpdateActive
type MovementRequest struct {
	petId    uint64
	movement model.Movement
	// keyPadStates is the client's per-move keypad history (see
	// petMoveKeyPadTail). Nil on versions without the tail.
	keyPadStates []byte
	// moveRect is the bounding rectangle of the whole path, appended after
	// the keypad block.
	moveRectLeft   int16
	moveRectTop    int16
	moveRectRight  int16
	moveRectBottom int16
}

func (m MovementRequest) PetId() uint64                { return m.petId }
func (m MovementRequest) PetIdAsUint32() uint32        { return uint32(m.petId) }
func (m MovementRequest) MovementData() model.Movement { return m.movement }
func (m MovementRequest) KeyPadStates() []byte         { return m.keyPadStates }

// MoveRect reports the path bounding rectangle the client appends after the
// keypad block: left, top, right, bottom.
func (m MovementRequest) MoveRect() (int16, int16, int16, int16) {
	return m.moveRectLeft, m.moveRectTop, m.moveRectRight, m.moveRectBottom
}

func (m MovementRequest) Operation() string {
	return PetMovementHandle
}

func (m MovementRequest) String() string {
	return fmt.Sprintf("petId [%d] elements [%d]", m.petId, len(m.movement.Elements))
}

func (m MovementRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if HasLeadingPetId(t) {
			w.WriteLong(m.petId) // absent on GMS v48 (single-pet)
		}
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		// Keypad history + path bounding rect — see petMoveKeyPadTail.
		if petMoveKeyPadTail(t) {
			w.WriteByte(byte(len(m.keyPadStates)))
			for i := 0; i < len(m.keyPadStates); i += 2 {
				b := m.keyPadStates[i] & 0x0F
				if i != len(m.keyPadStates)-1 {
					b |= m.keyPadStates[i+1] << 4
				}
				w.WriteByte(b)
			}
			w.WriteShort(uint16(m.moveRectLeft))
			w.WriteShort(uint16(m.moveRectTop))
			w.WriteShort(uint16(m.moveRectRight))
			w.WriteShort(uint16(m.moveRectBottom))
		}
		return w.Bytes()
	}
}

func (m *MovementRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if HasLeadingPetId(t) {
			m.petId = r.ReadUint64() // absent on GMS v48 (single-pet)
		}
		m.movement.Decode(l, ctx)(r, options)
		// Keypad history + path bounding rect, serverbound only — see
		// petMoveKeyPadTail.
		if petMoveKeyPadTail(t) {
			count := int(r.ReadByte())
			states := make([]byte, 0, count)
			var packed byte
			for i := 0; i < count; i++ {
				if i%2 == 0 {
					packed = r.ReadByte()
				} else {
					packed >>= 4
				}
				states = append(states, packed&0x0F)
			}
			m.keyPadStates = states
			m.moveRectLeft = r.ReadInt16()
			m.moveRectTop = r.ReadInt16()
			m.moveRectRight = r.ReadInt16()
			m.moveRectBottom = r.ReadInt16()
		}
	}
}
