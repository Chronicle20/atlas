package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/stat"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER"
	CommandCreateCharacter     = "CREATE_CHARACTER"
	CommandChangeMap           = "CHANGE_MAP"
	CommandChangeJob           = "CHANGE_JOB"
	CommandChangeHair          = "CHANGE_HAIR"
	CommandChangeFace          = "CHANGE_FACE"
	CommandChangeSkin          = "CHANGE_SKIN"
	CommandAwardExperience     = "AWARD_EXPERIENCE"
	CommandDeductExperience    = "DEDUCT_EXPERIENCE"
	CommandAwardLevel          = "AWARD_LEVEL"
	CommandRequestChangeMeso   = "REQUEST_CHANGE_MESO"
	CommandRequestDropMeso     = "REQUEST_DROP_MESO"
	CommandRequestChangeFame   = "REQUEST_CHANGE_FAME"
	CommandRequestDistributeAp = "REQUEST_DISTRIBUTE_AP"
	CommandRequestDistributeSp = "REQUEST_DISTRIBUTE_SP"
	CommandChangeHP            = "CHANGE_HP"
	CommandChangeMP            = "CHANGE_MP"
	CommandSetHP               = "SET_HP"
	CommandResetStats          = "RESET_STATS"
)

const (
	ExperienceDistributionTypeWhite        = "WHITE"
	ExperienceDistributionTypeYellow       = "YELLOW"
	ExperienceDistributionTypeChat         = "CHAT"
	ExperienceDistributionTypeMonsterBook  = "MONSTER_BOOK"
	ExperienceDistributionTypeMonsterEvent = "MONDEVENTSTER_"
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
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ChangeMapBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
}

type ChangeJobCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	JobId     job.Id     `json:"jobId"`
}

type ChangeHairCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	StyleId   uint32     `json:"styleId"`
}

type ChangeFaceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	StyleId   uint32     `json:"styleId"`
}

type ChangeSkinCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	StyleId   byte       `json:"styleId"`
}

type AwardExperienceCommandBody struct {
	ChannelId     channel.Id                `json:"channelId"`
	Distributions []ExperienceDistributions `json:"distributions"`
}

type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

type AwardLevelCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    byte       `json:"amount"`
}

type RequestChangeMesoBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int32  `json:"amount"`
}

type RequestDropMesoCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Amount    uint32     `json:"amount"`
}

type RequestChangeFameBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
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

type ChangeHPBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type ChangeMPBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type SetHPBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    uint16     `json:"amount"`
}

type ResetStatsCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
}

type DeductExperienceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    uint32     `json:"amount"`
}

type CreateCharacterCommandBody struct {
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	MaxHp        uint16   `json:"maxHp"`
	MaxMp        uint16   `json:"maxMp"`
	JobId        job.Id   `json:"jobId"`
	Gender       byte     `json:"gender"`
	Hair         uint32   `json:"hair"`
	Face         uint32   `json:"face"`
	SkinColor    byte     `json:"skinColor"`
	MapId        _map.Id  `json:"mapId"`
}

const (
	EnvEventTopicCharacterStatus     = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated           = "CREATED"
	StatusEventTypeLogin             = "LOGIN"
	StatusEventTypeLogout            = "LOGOUT"
	StatusEventTypeChannelChanged    = "CHANNEL_CHANGED"
	StatusEventTypeMapChanged        = "MAP_CHANGED"
	StatusEventTypeJobChanged        = "JOB_CHANGED"
	StatusEventTypeExperienceChanged = "EXPERIENCE_CHANGED"
	StatusEventTypeLevelChanged      = "LEVEL_CHANGED"
	StatusEventTypeMesoChanged       = "MESO_CHANGED"
	StatusEventTypeFameChanged       = "FAME_CHANGED"
	StatusEventTypeStatChanged       = "STAT_CHANGED"
	StatusEventTypeDeleted           = "DELETED"
	StatusEventTypeCreationFailed    = "CREATION_FAILED"

	StatusEventTypeError              = "ERROR"
	StatusEventErrorTypeNotEnoughMeso = "NOT_ENOUGH_MESO"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventCreatedBody struct {
	Name string `json:"name"`
}

type StatusEventCreationFailedBody struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type StatusEventLoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type StatusEventLogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type ChangeChannelEventLoginBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

type JobChangedStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	JobId     job.Id     `json:"jobId"`
}

type ExperienceChangedStatusEventBody struct {
	ChannelId     channel.Id                `json:"channelId"`
	Current       uint32                    `json:"current"`
	Distributions []ExperienceDistributions `json:"distributions"`
}

type LevelChangedStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    byte       `json:"amount"`
	Current   byte       `json:"current"`
}

type StatusEventDeletedBody struct {
}

type StatusEventErrorBody[F any] struct {
	Error string `json:"error"`
	Body  F      `json:"body"`
}

type MesoChangedStatusEventBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int32  `json:"amount"`
}

type NotEnoughMesoErrorStatusBodyBody struct {
	Amount int32 `json:"amount"`
}

// StatusEventMesoErrorBody is a non-generic error body for meso-related errors
// This avoids nested generic type issues with Kafka message deserialization
type StatusEventMesoErrorBody struct {
	Error  string `json:"error"`
	Amount int32  `json:"amount"`
}

type FameChangedStatusEventBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}

type StatusEventStatChangedBody struct {
	ChannelId       channel.Id             `json:"channelId"`
	ExclRequestSent bool                   `json:"exclRequestSent"`
	Updates         []stat.Type            `json:"updates"`
	Values          map[string]interface{} `json:"values,omitempty"`
}
