package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonMoveWriter = "DragonMove"

// DragonMove is the server -> client MOVE_DRAGON packet: int ownerCharacterId
// (consumed upstream by CUserPool::OnUserCommonPacket) followed by the raw
// CMovePath blob, rebroadcast byte-faithfully.
//
// CDragon::OnMove (GMS v95.0 @0x50ad30) is a single call:
// CMovePath::OnMovePacket(&m_pvc[142], iPacket, 0). The whole body is the blob.
// The blob already begins with the start position (CMovePath::Encode writes
// Encode2 startX, Encode2 startY first), so the start position must NOT be
// written separately — doing so makes the observing client's CMovePath::Decode
// read 4 bytes off and throw.
//
// Layout is identical across all six applicable versions. No version gate.
//
// packet-audit:fname CDragon::OnMove
type DragonMove struct {
	ownerCharacterId uint32
	rawMovement      []byte
}

func NewDragonMove(ownerCharacterId uint32, rawMovement []byte) DragonMove {
	return DragonMove{ownerCharacterId: ownerCharacterId, rawMovement: rawMovement}
}

func (m DragonMove) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonMove) RawMovement() []byte      { return m.rawMovement }
func (m DragonMove) Operation() string        { return DragonMoveWriter }
func (m DragonMove) String() string {
	return fmt.Sprintf("ownerCharacterId [%d], rawMovement [%d bytes]", m.ownerCharacterId, len(m.rawMovement))
}

func (m DragonMove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		w.WriteByteArray(m.rawMovement) // CMovePath blob — begins with start x,y
		return w.Bytes()
	}
}

func (m *DragonMove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
		m.rawMovement = r.ReadBytes(r.Available())
	}
}
