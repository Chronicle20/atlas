package serverbound

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonMoveHandle = "DragonMoveHandle"

// Move is the client -> server MOVE_DRAGON packet, decoded from the real client
// SEND site CVecCtrlDragon::EndUpdateActive (GMS v95.0 @0x996570, v83
// @0x9b7b9c). The send is exactly:
//
//	COutPacket(op)
//	CMovePath::Flush(...)   ; the opaque movement blob
//
// THERE IS NO LEADING IDENTITY FIELD. Every sibling move packet in this
// codebase has one — CVecCtrlSummoned::EndUpdateActive writes Encode4 summonId
// before the blob — so its absence here looks like a bug and is not. The dragon
// is 1:1 with its owning CUser, so the server resolves it entirely from the
// sending session's character id. A consequence worth naming: "naming a dragon
// the submitter does not own" is unrepresentable on the wire.
//
// The CMovePath blob is not trivially parseable without a full move-path codec,
// so the whole body is treated as opaque and rebroadcast byte-faithfully.
// startX/startY are lifted from its first 4 bytes (CMovePath::Encode leads with
// Encode2 startX, Encode2 startY) only to seed the persisted position.
//
// Layout is identical across all six applicable versions. No version gate.
//
// packet-audit:fname CVecCtrlDragon::EndUpdateActive
type Move struct {
	startX      int16
	startY      int16
	rawMovement []byte
}

func (m Move) StartX() int16       { return m.startX }
func (m Move) StartY() int16       { return m.startY }
func (m Move) RawMovement() []byte { return m.rawMovement }
func (m Move) Operation() string   { return DragonMoveHandle }
func (m Move) String() string {
	return fmt.Sprintf("startX [%d], startY [%d], rawMovement [%d bytes]", m.startX, m.startY, len(m.rawMovement))
}

func (m Move) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.rawMovement)
		return w.Bytes()
	}
}

func (m *Move) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.rawMovement = r.ReadBytes(r.Available())
		if len(m.rawMovement) >= 4 {
			m.startX = int16(binary.LittleEndian.Uint16(m.rawMovement[0:2]))
			m.startY = int16(binary.LittleEndian.Uint16(m.rawMovement[2:4]))
		}
	}
}
