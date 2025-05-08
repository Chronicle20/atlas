package character

const (
	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"
	CommandChangeJob          = "CHANGE_JOB"
	CommandAwardExperience    = "AWARD_EXPERIENCE"
	CommandAwardLevel         = "AWARD_LEVEL"
	CommandRequestChangeMeso  = "REQUEST_CHANGE_MESO"

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
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type ChangeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}

type ChangeJobCommandBody struct {
	ChannelId byte   `json:"channelId"`
	JobId     uint16 `json:"jobId"`
}

type AwardExperienceCommandBody struct {
	ChannelId     byte                      `json:"channelId"`
	Distributions []ExperienceDistributions `json:"distributions"`
}

type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

type AwardLevelCommandBody struct {
	ChannelId byte `json:"channelId"`
	Amount    byte `json:"amount"`
}

type RequestChangeMesoBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int32  `json:"amount"`
}
