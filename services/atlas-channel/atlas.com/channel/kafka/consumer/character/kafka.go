package character

const (
	EnvEventTopicCharacterStatus     = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeMapChanged        = "MAP_CHANGED"
	StatusEventTypeJobChanged        = "JOB_CHANGED"
	StatusEventTypeExperienceChanged = "EXPERIENCE_CHANGED"
	StatusEventTypeLevelChanged      = "LEVEL_CHANGED"
	StatusEventTypeMesoChanged       = "MESO_CHANGED"
	StatusEventTypeFameChanged       = "FAME_CHANGED"
	StatusEventTypeStatChanged       = "STAT_CHANGED"

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

	StatusEventActorTypeCharacter = "CHARACTER"
)

type statusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

type statusEventStatChangedBody struct {
	ChannelId       byte     `json:"channelId"`
	ExclRequestSent bool     `json:"exclRequestSent"`
	Updates         []string `json:"updates"`
}

type statusEventMapChangedBody struct {
	ChannelId      byte   `json:"channelId"`
	OldMapId       uint32 `json:"oldMapId"`
	TargetMapId    uint32 `json:"targetMapId"`
	TargetPortalId uint32 `json:"targetPortalId"`
}

type experienceChangedStatusEventBody struct {
	ChannelId     byte                      `json:"channelId"`
	Current       uint32                    `json:"current"`
	Distributions []experienceDistributions `json:"distributions"`
}

type experienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

type levelChangedStatusEventBody struct {
	ChannelId byte `json:"channelId"`
	Amount    byte `json:"amount"`
	Current   byte `json:"current"`
}

type fameChangedStatusEventBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}

type mesoChangedStatusEventBody struct {
	Amount int32 `json:"amount"`
}
