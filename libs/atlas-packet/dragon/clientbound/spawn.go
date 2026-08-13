package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonSpawnWriter = "DragonSpawn"

// DragonSpawn is the server -> client SPAWN_DRAGON packet.
//
// Wire: int ownerCharacterId, int x, int y, byte stance, short <discarded>,
// short jobId.
//
// The leading ownerCharacterId is consumed by CUserPool::OnUserCommonPacket
// (GMS v95.0 @0x94cdb0), which resolves the CUser and only then routes the
// dragon family. The dragon has no wire identity of its own.
//
// TWO TRAPS, both load-bearing:
//   - x and y are FOUR bytes each (Decode4), not the 2-byte coordinates used by
//     every other entity in the protocol.
//   - the short between stance and jobId is read by the client and thrown away
//     (the Decode2 return value is never assigned). It must still be written or
//     jobId is read from the wrong offset.
//
// Layout is identical across v83/v84/v87/v92/v95/JMS185 — verified in two
// distinct client size classes (0x330: v83/v87/JMS185, 0x464: v92/v95). No
// version gate.
//
// packet-audit:fname CDragon::OnCreated
type DragonSpawn struct {
	ownerCharacterId uint32
	x                int32
	y                int32
	stance           byte
	jobId            uint16
}

func NewDragonSpawn(ownerCharacterId uint32, x int32, y int32, stance byte, jobId uint16) DragonSpawn {
	return DragonSpawn{ownerCharacterId: ownerCharacterId, x: x, y: y, stance: stance, jobId: jobId}
}

func (m DragonSpawn) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonSpawn) X() int32                 { return m.x }
func (m DragonSpawn) Y() int32                 { return m.y }
func (m DragonSpawn) Stance() byte             { return m.stance }
func (m DragonSpawn) JobId() uint16            { return m.jobId }
func (m DragonSpawn) Operation() string        { return DragonSpawnWriter }
func (m DragonSpawn) String() string {
	return fmt.Sprintf("ownerCharacterId [%d], x [%d], y [%d], stance [%d], jobId [%d]", m.ownerCharacterId, m.x, m.y, m.stance, m.jobId)
}

func (m DragonSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		w.WriteByte(m.stance)
		w.WriteShort(0) // client decodes and discards; omitting it misaligns jobId
		w.WriteShort(m.jobId)
		return w.Bytes()
	}
}

func (m *DragonSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
		m.stance = r.ReadByte()
		_ = r.ReadUint16() // discarded by the client (see Encode)
		m.jobId = r.ReadUint16()
	}
}
