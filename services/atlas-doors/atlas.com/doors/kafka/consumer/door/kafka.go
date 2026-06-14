package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const EnvCommandTopic = "COMMAND_TOPIC_DOOR"

const (
	CommandTypeSpawn  = "SPAWN"
	CommandTypeRemove = "REMOVE"
)

type Command[E any] struct {
	WorldId          world.Id     `json:"worldId"`
	ChannelId        channel.Id   `json:"channelId"`
	MapId            _map.Id      `json:"mapId"`
	Instance         uuid.UUID    `json:"instance"`
	OwnerCharacterId character.Id `json:"ownerCharacterId"`
	Type             string       `json:"type"`
	Body             E            `json:"body"`
}

type SpawnBody struct {
	SkillId    skill.Id `json:"skillId"`
	SkillLevel byte     `json:"skillLevel"`
	X          point.X  `json:"x"`
	Y          point.Y  `json:"y"`
}

type RemoveBody struct {
	Reason string `json:"reason"`
}
