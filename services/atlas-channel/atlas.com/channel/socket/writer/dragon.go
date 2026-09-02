package writer

import (
	dragoncb "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// DragonSpawnBody builds the SPAWN_DRAGON packet for an Evan's dragon,
// broadcast to every session in the owner's map INCLUDING the owner.
// x/y are int32: this packet encodes 4-byte coordinates.
func DragonSpawnBody(ownerCharacterId uint32, x int32, y int32, stance byte, jobId uint16) packet.Encode {
	return dragoncb.NewDragonSpawn(ownerCharacterId, x, y, stance, jobId).Encode
}

// DragonMoveBody builds the MOVE_DRAGON packet, rebroadcasting the raw CMovePath
// blob to OTHER sessions. The blob is NOT echoed verbatim: dragoncb.DragonMove
// re-serializes it at encode time for the RECEIVING client, because GMS v87 reads
// the per-element XOffset/YOffset pair the client sends but is never sent it back
// (CMovePath::Encode @0x6c70fe writes the pair, ::Decode @0x6c6e86 never reads
// it). The blob already carries the start position, so it is not written
// separately.
func DragonMoveBody(ownerCharacterId uint32, rawMovement []byte) packet.Encode {
	return dragoncb.NewDragonMove(ownerCharacterId, rawMovement).Encode
}

// DragonRemoveBody builds the REMOVE_DRAGON packet. The client has no handler
// arm for this opcode and discards it — the dragon disappears because the
// owner's CUser is destroyed when they leave the field. Sending it is correct
// and harmless, but it is not the removal mechanism. See dragoncb.DragonRemove.
func DragonRemoveBody(ownerCharacterId uint32) packet.Encode {
	return dragoncb.NewDragonRemove(ownerCharacterId).Encode
}
