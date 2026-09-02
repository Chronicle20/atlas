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

const NPCActionHandle = "NPCActionHandle"

// npcMoveKeyPadTail reports whether the serverbound NPC_ACTION movement blob
// ends with the keypad history + path bounding rectangle that
// CMovePath::Encode appends after the element loop (see moveKeyPadTail in
// character/serverbound/move.go for the full field derivation:
// count-of-entries byte, ceil(count/2) packed nibble bytes, then left/top/
// right/bottom int16).
//
// Live-confirmed for jms v185: `[PKT IN] handler=NPCActionHandle op=0x00d0
// len=40` captured on the atlas-main k3s environment decodes objectId=2,
// unk=255, unk2=255, movement startX=833/startY=125/1 NORMAL element, and
// leaves exactly 9 unread trailing bytes — `00 | 41 03 7d 00 | 41 03 7d 00`,
// i.e. keypad count=0 followed by the single-point bounding rect (833,125,
// 833,125) that a zero-velocity one-element path produces. See
// docs/tasks/fix-jms185-attack-decode/sibling-movement-ops-findings.md §1.
//
// Sender is CNpc::GenerateMovePath @0x7199ce, which matches Atlas's header
// (Encode4(dwNpcId), Encode1(nAction), Encode1(nChatIdx)) and delegates the
// movement body to CMovePath::Flush @0x70ba2c, the same encoder that
// produces the character-move tail — unconditionally, regardless of the
// pbPassive argument Flush is called with.
//
// Gated to JMS because that is the only client this sender was read on. The
// GMS senders were not read for this op; do not extend this gate to GMS
// without reading GMS's CNpc::GenerateMovePath directly.
func npcMoveKeyPadTail(t tenant.Model) bool {
	return t.IsRegion("JMS")
}

// packet-audit:fname CNpc::GenerateMovePath
type ActionRequest struct {
	objectId    uint32
	unk         byte
	unk2        byte
	hasMovement bool
	movement    model.Movement
	// keyPadStates is the client's per-move keypad history (see
	// npcMoveKeyPadTail). Nil on versions without the tail, or when
	// hasMovement is false.
	keyPadStates []byte
	// moveRect is the bounding rectangle of the whole path, appended after
	// the keypad block.
	moveRectLeft   int16
	moveRectTop    int16
	moveRectRight  int16
	moveRectBottom int16
}

func (m ActionRequest) ObjectId() uint32             { return m.objectId }
func (m ActionRequest) Unk() byte                    { return m.unk }
func (m ActionRequest) Unk2() byte                   { return m.unk2 }
func (m ActionRequest) HasMovement() bool            { return m.hasMovement }
func (m ActionRequest) MovementData() model.Movement { return m.movement }
func (m ActionRequest) KeyPadStates() []byte         { return m.keyPadStates }

// MoveRect reports the path bounding rectangle the client appends after the
// keypad block: left, top, right, bottom.
func (m ActionRequest) MoveRect() (int16, int16, int16, int16) {
	return m.moveRectLeft, m.moveRectTop, m.moveRectRight, m.moveRectBottom
}

func (m ActionRequest) Operation() string {
	return NPCActionHandle
}

func (m ActionRequest) String() string {
	if m.hasMovement {
		return fmt.Sprintf("objectId [%d] unk [%d] unk2 [%d] hasMovement [true] elements [%d]", m.objectId, m.unk, m.unk2, len(m.movement.Elements))
	}
	return fmt.Sprintf("objectId [%d] unk [%d] unk2 [%d] hasMovement [false]", m.objectId, m.unk, m.unk2)
}

func (m ActionRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.objectId)
		w.WriteByte(m.unk)
		w.WriteByte(m.unk2)
		if m.hasMovement {
			w.WriteByteArray(m.movement.Encode(l, ctx)(options))
			// Keypad history + path bounding rect — see npcMoveKeyPadTail.
			if npcMoveKeyPadTail(t) {
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
		}
		return w.Bytes()
	}
}

func (m *ActionRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.objectId = r.ReadUint32()
		m.unk = r.ReadByte()
		m.unk2 = r.ReadByte()
		if r.Available() > 0 {
			m.hasMovement = true
			m.movement.Decode(l, ctx)(r, options)
			// Keypad history + path bounding rect, serverbound only — see
			// npcMoveKeyPadTail.
			if npcMoveKeyPadTail(t) {
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
}
