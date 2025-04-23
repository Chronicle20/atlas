package pet

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PET"
	CommandPetSpawn          = "SPAWN"
	CommandPetDespawn        = "DESPAWN"
	CommandPetAttemptCommand = "ATTEMPT_COMMAND"
	CommandAwardCloseness    = "AWARD_CLOSENESS"
	CommandAwardFullness     = "AWARD_FULLNESS"
	CommandAwardLevel        = "AWARD_LEVEL"
	CommandSetExclude        = "EXCLUDE"
)

type command[E any] struct {
	ActorId uint32 `json:"actorId"`
	PetId   uint32 `json:"petId"`
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

type setExcludeCommandBody struct {
	Items []uint32 `json:"items"`
}

const (
	EnvCommandTopicMovement = "COMMAND_TOPIC_PET_MOVEMENT"
)

type movementCommand struct {
	WorldId    byte   `json:"worldId"`
	ChannelId  byte   `json:"channelId"`
	MapId      uint32 `json:"mapId"`
	ObjectId   uint64 `json:"objectId"`
	ObserverId uint32 `json:"observerId"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
}
