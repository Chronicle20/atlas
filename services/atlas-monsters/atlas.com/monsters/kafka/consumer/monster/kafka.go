package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage = "DAMAGE"

	EnvCommandTopicMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
)

type command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type damageCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

type movementCommand struct {
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      _map.Id    `json:"mapId"`
	Instance   uuid.UUID  `json:"instance"`
	ObjectId   uint64     `json:"objectId"`
	ObserverId uint32     `json:"observerId"`
	X          int16      `json:"x"`
	Y          int16      `json:"y"`
	Stance     byte       `json:"stance"`
}
