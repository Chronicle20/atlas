package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const RemoveDoorWriter = "RemoveDoor"

// RemoveDoor is the clientbound packet that despawns a Mystic Door from the
// field (the non-town/area-door path).
//
// the removeDoor, town=false branch:
//
//	p.writeByte(0) — constant zero byte (client expects this discriminant)
//	p.writeInt(ownerId) — LE uint32 door owner character id
//
// The town=true branch in the v83 client emits SPAWN_PORTAL with two NONE map-ids
// (999999999); that is a distinct packet modelled by SpawnPortal with
// _map.Id(999999999) arguments — it uses a different config writer name and
// opcode, so it is NOT represented here.
//
// Layout is identical across all tenant versions (no structural delta found);
// branching is deferred to Part H if one appears.
type RemoveDoor struct {
	ownerId uint32
}

// NewRemoveDoor constructs a RemoveDoor packet for the area-door (non-town) path.
func NewRemoveDoor(ownerId uint32) RemoveDoor {
	return RemoveDoor{ownerId: ownerId}
}

func (m RemoveDoor) OwnerId() uint32   { return m.ownerId }
func (m RemoveDoor) Operation() string { return RemoveDoorWriter }
func (m RemoveDoor) String() string {
	return fmt.Sprintf("RemoveDoor{ownerId=%d}", m.ownerId)
}

// Encode encodes the removeDoor (non-town) body (no opcode — config-driven at runtime).
// Layout: writeByte(0), writeInt(ownerId).
func (m RemoveDoor) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteInt(m.ownerId)
		return w.Bytes()
	}
}
