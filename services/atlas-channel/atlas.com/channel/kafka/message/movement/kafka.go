package movement

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandMonsterMovement   = "COMMAND_TOPIC_MONSTER_MOVEMENT"
	EnvCommandCharacterMovement = "COMMAND_TOPIC_CHARACTER_MOVEMENT"
	EnvCommandPetMovement       = "COMMAND_TOPIC_PET_MOVEMENT"
)

type Command[E any] struct {
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      _map.Id    `json:"mapId"`
	ObjectId   uint64 `json:"objectId"`
	ObserverId uint32 `json:"observerId"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
}
