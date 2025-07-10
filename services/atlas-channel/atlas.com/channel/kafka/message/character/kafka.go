package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER"
	CommandRequestDistributeAp = "REQUEST_DISTRIBUTE_AP"
	CommandRequestDistributeSp = "REQUEST_DISTRIBUTE_SP"
	CommandRequestDropMeso     = "REQUEST_DROP_MESO"
	CommandChangeHP            = "CHANGE_HP"
	CommandChangeMP            = "CHANGE_MP"

	CommandDistributeApAbilityStrength     = "STRENGTH"
	CommandDistributeApAbilityDexterity    = "DEXTERITY"
	CommandDistributeApAbilityIntelligence = "INTELLIGENCE"
	CommandDistributeApAbilityLuck         = "LUCK"
	CommandDistributeApAbilityHp           = "HP"
	CommandDistributeApAbilityMp           = "MP"
)

type Command[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type DistributePair struct {
	Ability string `json:"ability"`
	Amount  int8   `json:"amount"`
}

type RequestDistributeApCommandBody struct {
	Distributions []DistributePair `json:"distributions"`
}

type RequestDistributeSpCommandBody struct {
	SkillId uint32 `json:"skilId"`
	Amount  int8   `json:"amount"`
}

type RequestDropMesoCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Amount    uint32     `json:"amount"`
}

type ChangeHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type ChangeMPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

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

type StatusEvent[E any] struct {
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	WorldId     world.Id `json:"worldId"`
	Body        E        `json:"body"`
}

type StatusEventStatChangedBody struct {
	ChannelId       channel.Id `json:"channelId"`
	ExclRequestSent bool       `json:"exclRequestSent"`
	Updates         []string   `json:"updates"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

type ExperienceChangedStatusEventBody struct {
	ChannelId     channel.Id                `json:"channelId"`
	Current       uint32                    `json:"current"`
	Distributions []ExperienceDistributions `json:"distributions"`
}

type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

type LevelChangedStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    byte       `json:"amount"`
	Current   byte       `json:"current"`
}

type FameChangedStatusEventBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}

type MesoChangedStatusEventBody struct {
	Amount int32 `json:"amount"`
}
