package pet

import "atlas-pets/pet"

const (
	EnvCommandTopicMovement   = "COMMAND_TOPIC_PET_MOVEMENT"
	MovementTypeNormal        = "NORMAL"
	MovementTypeTeleport      = "TELEPORT"
	MovementTypeStartFallDown = "START_FALL_DOWN"
	MovementTypeFlyingBlock   = "FLYING_BLOCK"
	MovementTypeJump          = "JUMP"
	MovementTypeStatChange    = "STAT_CHANGE"
)

type movementCommand struct {
	WorldId     byte         `json:"worldId"`
	ChannelId   byte         `json:"channelId"`
	MapId       uint32       `json:"mapId"`
	PetId       uint32       `json:"petId"`
	CharacterId uint32       `json:"characterId"`
	Movement    pet.Movement `json:"movement"`
}
