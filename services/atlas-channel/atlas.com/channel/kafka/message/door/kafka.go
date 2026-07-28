package door

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const EnvDoorCommandTopic = "COMMAND_TOPIC_DOOR"

const (
	CommandTypeSpawn  = "SPAWN"
	CommandTypeRemove = "REMOVE"
)

type Command[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

type SpawnBody struct {
	SkillId    uint32 `json:"skillId"`
	SkillLevel byte   `json:"skillLevel"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
}

type RemoveBody struct {
	Reason string `json:"reason"`
}
