package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpawnDoorWriter = "SpawnDoor"

// SpawnDoor is the clientbound packet that spawns a Mystic Door on the field.
//
// Cosmic PacketCreator.java spawnDoor (line 1112):
//
//	p.writeBool(launched)    — byte: 1=already deployed, 0=first spawn
//	p.writeInt(ownerid)      — LE uint32
//	p.writePos(pos)          — writeShort(x), writeShort(y)  [ByteBufOutPacket line 85-87]
//
// Total: 9 bytes. Layout is identical across all tenant versions (no structural
// delta found in any IDA audit); branching is deferred to Part H if one appears.
type SpawnDoor struct {
	ownerId  uint32
	x        int16
	y        int16
	launched bool
}

// NewSpawnDoor constructs a SpawnDoor packet.
// launched: true if the door is being shown to a player who logs in/warps in
// after the door was already deployed (vs first-spawn for everyone present).
func NewSpawnDoor(ownerId uint32, x, y int16, launched bool) SpawnDoor {
	return SpawnDoor{ownerId: ownerId, x: x, y: y, launched: launched}
}

func (m SpawnDoor) OwnerId() uint32  { return m.ownerId }
func (m SpawnDoor) X() int16         { return m.x }
func (m SpawnDoor) Y() int16         { return m.y }
func (m SpawnDoor) Launched() bool   { return m.launched }
func (m SpawnDoor) Operation() string { return SpawnDoorWriter }
func (m SpawnDoor) String() string {
	return fmt.Sprintf("SpawnDoor{ownerId=%d x=%d y=%d launched=%t}", m.ownerId, m.x, m.y, m.launched)
}

// Encode encodes the spawnDoor body (no opcode — config-driven at runtime).
// Layout: writeBool(launched), writeInt(ownerId), writeShort(x), writeShort(y).
func (m SpawnDoor) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.launched)
		w.WriteInt(m.ownerId)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}
