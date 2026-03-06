package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	MiniRoom = "MiniRoom"
)

func MiniRoomSpawnBody(characterId uint32, mr model.MiniRoom) packet.Encode {
	return mr.Spawn(characterId)
}

func MiniRoomDespawnBody(characterId uint32, mr model.MiniRoom) packet.Encode {
	return mr.Despawn(characterId)
}
