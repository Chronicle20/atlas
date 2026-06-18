package clientbound

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const RemoveTownDoorWriter = "RemoveTownDoor"

// RemoveTownDoor is the clientbound packet that clears the minimap door indicator
// on the town side when a Mystic Door is removed.
//
// the removeDoor, town=true branch:
//
//	p = OutPacket.create(SendOpcode.SPAWN_PORTAL)
//	p.writeInt(MapId.NONE) — LE uint32, 999999999 = _map.EmptyMapId
//	p.writeInt(MapId.NONE) — LE uint32, 999999999 = _map.EmptyMapId
//
// Unlike SpawnPortal (which places a real minimap portal and always writes a
// position), this removal packet writes EXACTLY two ints — NO position field.
// Total body: 8 bytes.
//
// Wire opcode = SPAWN_PORTAL (same as SpawnPortal); Part H maps both writer
// names (SpawnPortalWriter and RemoveTownDoorWriter) to that opcode.
//
// IMPORTANT: Do NOT use SpawnPortal(MapNone, MapNone, 0, 0) for town-side
// removal. SpawnPortal always writes position (12 bytes); this packet must be
// 8 bytes. Using SpawnPortal for removal emits 4 spurious trailing bytes and
// corrupts the client read cursor.
//
// Layout is version-unbranched (single the v83 client layout applies to all tenants;
// branching is deferred to Part H if a structural delta is found).
type RemoveTownDoor struct{}

// NewRemoveTownDoor constructs a RemoveTownDoor packet. No arguments are
// needed — the v83 client always sends _map.EmptyMapId (999999999) for both map ids
// on the town-side removal path.
func NewRemoveTownDoor() RemoveTownDoor {
	return RemoveTownDoor{}
}

func (m RemoveTownDoor) Operation() string { return RemoveTownDoorWriter }
func (m RemoveTownDoor) String() string    { return "RemoveTownDoor{}" }

// Encode encodes the town-side door removal body (no opcode — config-driven at runtime).
// Layout: writeInt(_map.EmptyMapId), writeInt(_map.EmptyMapId) — 8 bytes total, no position.
func (m RemoveTownDoor) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(_map.EmptyMapId))
		w.WriteInt(uint32(_map.EmptyMapId))
		return w.Bytes()
	}
}
