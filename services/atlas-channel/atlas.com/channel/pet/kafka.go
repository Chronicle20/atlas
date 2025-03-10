package pet

import "atlas-channel/movement"

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
