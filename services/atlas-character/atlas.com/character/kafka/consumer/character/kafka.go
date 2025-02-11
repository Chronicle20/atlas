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

	ExperienceDistributionTypeWhite        = "WHITE"
	ExperienceDistributionTypeYellow       = "YELLOW"
	ExperienceDistributionTypeChat         = "CHAT"
	ExperienceDistributionTypeMonsterBook  = "MONSTER_BOOK"
	ExperienceDistributionTypeMonsterEvent = "MONSTER_EVENT"
	ExperienceDistributionTypePlayTime     = "PLAY_TIME"
	ExperienceDistributionTypeWedding      = "WEDDING"
	ExperienceDistributionTypeSpiritWeek   = "SPIRIT_WEEK"
	ExperienceDistributionTypeParty        = "PARTY"
	ExperienceDistributionTypeItem         = "ITEM"
	ExperienceDistributionTypeInternetCafe = "INTERNET_CAFE"
	ExperienceDistributionTypeRainbowWeek  = "RAINBOW_WEEK"
	ExperienceDistributionTypePartyRing    = "PARTY_RING"
	ExperienceDistributionTypeCakePie      = "CAKE_PIE"

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
	ChannelId     byte                      `json:"channelId"`
	Distributions []experienceDistributions `json:"distributions"`
}

type experienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
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
