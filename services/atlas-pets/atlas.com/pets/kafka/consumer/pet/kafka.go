package pet

import "atlas-pets/pet"

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PET"
	CommandPetSpawn          = "SPAWN"
	CommandPetDespawn        = "DESPAWN"
	CommandPetAttemptCommand = "ATTEMPT_COMMAND"
	CommandAwardCloseness    = "AWARD_CLOSENESS"
	CommandAwardFullness     = "AWARD_FULLNESS"
	CommandAwardLevel        = "AWARD_LEVEL"
)

type command[E any] struct {
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

type attemptCommandCommandBody struct {
	CommandId byte `json:"commandId"`
	ByName    bool `json:"byName"`
}

type awardClosenessCommandBody struct {
	Amount uint16 `json:"amount"`
}

type awardFullnessCommandBody struct {
	Amount byte `json:"amount"`
}

type awardLevelCommandBody struct {
	Amount byte `json:"amount"`
}

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
	PetId       uint64       `json:"petId"`
	CharacterId uint32       `json:"characterId"`
	Movement    pet.Movement `json:"movement"`
}
