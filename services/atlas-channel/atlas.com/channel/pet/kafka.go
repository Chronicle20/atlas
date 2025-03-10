package pet

import "atlas-channel/movement"

const (
	EnvCommandTopic   = "COMMAND_TOPIC_PET"
	CommandPetSpawn   = "SPAWN"
	CommandPetDespawn = "DESPAWN"
)

type commandEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	PetId   uint64 `json:"petId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type spawnCommandBody struct {
	Lead bool `json:"lead"`
}

type despawnCommandBody struct {
}

const (
	EnvCommandMovement = "COMMAND_TOPIC_PET_MOVEMENT"
)

type movementCommand struct {
	WorldId     byte              `json:"worldId"`
	ChannelId   byte              `json:"channelId"`
	MapId       uint32            `json:"mapId"`
	PetId       uint64            `json:"petId"`
	CharacterId uint32            `json:"characterId"`
	Movement    movement.Movement `json:"movement"`
}
