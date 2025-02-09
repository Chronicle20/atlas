package character

import "atlas-character/character"

const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER"
	CommandChangeMap           = "CHANGE_MAP"
	CommandChangeJob           = "CHANGE_JOB"
	CommandAwardExperience     = "AWARD_EXPERIENCE"
	CommandAwardLevel          = "AWARD_LEVEL"
	CommandRequestChangeMeso   = "REQUEST_CHANGE_MESO"
	CommandRequestDropMeso     = "REQUEST_DROP_MESO"
	CommandRequestChangeFame   = "REQUEST_CHANGE_FAME"
	CommandRequestDistributeAp = "REQUEST_DISTRIBUTE_AP"

	EnvCommandTopicMovement   = "COMMAND_TOPIC_CHARACTER_MOVEMENT"
	MovementTypeNormal        = "NORMAL"
	MovementTypeTeleport      = "TELEPORT"
	MovementTypeStartFallDown = "START_FALL_DOWN"
	MovementTypeFlyingBlock   = "FLYING_BLOCK"
	MovementTypeJump          = "JUMP"
	MovementTypeStatChange    = "STAT_CHANGE"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type changeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}

type changeJobCommandBody struct {
	ChannelId byte   `json:"channelId"`
	JobId     uint16 `json:"jobId"`
}

type awardExperienceCommandBody struct {
	ChannelId byte   `json:"channelId"`
	Amount    uint32 `json:"amount"`
}

type awardLevelCommandBody struct {
	ChannelId byte `json:"channelId"`
	Amount    byte `json:"amount"`
}

type requestChangeMesoBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int32  `json:"amount"`
}

type requestDropMesoCommandBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Amount    uint32 `json:"amount"`
}

type requestChangeFameBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}

type DistributePair struct {
	Ability string `json:"ability"`
	Amount  int8   `json:"amount"`
}

type requestDistributeApCommandBody struct {
	Distributions []DistributePair `json:"distributions"`
}

type movementCommand struct {
	WorldId     byte               `json:"worldId"`
	ChannelId   byte               `json:"channelId"`
	MapId       uint32             `json:"mapId"`
	CharacterId uint32             `json:"characterId"`
	Movement    character.Movement `json:"movement"`
}
