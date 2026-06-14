package clientbound

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpawnPortalWriter = "SpawnPortal"

// SpawnPortal is the clientbound packet that places the minimap door indicator
// (SPAWN_PORTAL) showing the town↔target portal link to the caster.
//
// Cosmic PacketCreator.java spawnPortal (line 1096):
//
//	p.writeInt(townId)      — LE uint32 town map id
//	p.writeInt(targetId)    — LE uint32 field (area) map id
//	p.writePos(pos)         — writeShort(x), writeShort(y)  [ByteBufOutPacket line 85-87]
//
// This encoder also covers the Cosmic removeDoor town=true path, which uses
// the SPAWN_PORTAL opcode with MapId.NONE (999999999) for both map ids and
// no position; callers model that as NewSpawnPortal(MapNone, MapNone, 0, 0).
//
// Total: 12 bytes. Layout is identical across all tenant versions (no
// structural delta found); branching is deferred to Part H if one appears.
type SpawnPortal struct {
	townMapId   _map.Id
	targetMapId _map.Id
	x           int16
	y           int16
}

// NewSpawnPortal constructs a SpawnPortal packet.
// townMapId/targetMapId: the paired map ids for the portal link.
// x, y: the minimap position of the portal indicator.
func NewSpawnPortal(townMapId, targetMapId _map.Id, x, y int16) SpawnPortal {
	return SpawnPortal{townMapId: townMapId, targetMapId: targetMapId, x: x, y: y}
}

func (m SpawnPortal) TownMapId() _map.Id   { return m.townMapId }
func (m SpawnPortal) TargetMapId() _map.Id { return m.targetMapId }
func (m SpawnPortal) X() int16             { return m.x }
func (m SpawnPortal) Y() int16             { return m.y }
func (m SpawnPortal) Operation() string    { return SpawnPortalWriter }
func (m SpawnPortal) String() string {
	return fmt.Sprintf("SpawnPortal{townMapId=%d targetMapId=%d x=%d y=%d}", m.townMapId, m.targetMapId, m.x, m.y)
}

// Encode encodes the spawnPortal body (no opcode — config-driven at runtime).
// Layout: writeInt(townMapId), writeInt(targetMapId), writeShort(x), writeShort(y).
func (m SpawnPortal) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(m.townMapId))
		w.WriteInt(uint32(m.targetMapId))
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}
