package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_CHARACTER"
)

const (
	CommandRequestDistributeAp    = "REQUEST_DISTRIBUTE_AP"
	CommandRequestDistributeSp    = "REQUEST_DISTRIBUTE_SP"
	CommandRequestDropMeso        = "REQUEST_DROP_MESO"
	CommandRequestChangeMeso      = "REQUEST_CHANGE_MESO"
	CommandChangeHP               = "CHANGE_HP"
	CommandChangeMP               = "CHANGE_MP"
	CommandSetHP                  = "SET_HP"
	CommandAwardExperience        = "AWARD_EXPERIENCE"
	CommandRedeemStoredExperience = "REDEEM_STORED_EXPERIENCE"

	CommandDistributeApAbilityStrength     = "STRENGTH"
	CommandDistributeApAbilityDexterity    = "DEXTERITY"
	CommandDistributeApAbilityIntelligence = "INTELLIGENCE"
	CommandDistributeApAbilityLuck         = "LUCK"
	CommandDistributeApAbilityHp           = "HP"
	CommandDistributeApAbilityMp           = "MP"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
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

type RequestChangeMesoBody struct {
	ActorId    uint32 `json:"actorId"`
	ActorType  string `json:"actorType"`
	Amount     int32  `json:"amount"`
	ShowEffect bool   `json:"showEffect"`
}

type ChangeHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type ChangeMPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type SetHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    uint16     `json:"amount"`
}

type AwardExperienceCommandBody struct {
	ChannelId     channel.Id                `json:"channelId"`
	Distributions []ExperienceDistributions `json:"distributions"`
	ShowEffect    bool                      `json:"showEffect"`
}

type RedeemStoredExperienceCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
}

const (
	EnvEventTopicCharacterStatus topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
)

const (
	StatusEventTypeMapChanged        = "MAP_CHANGED"
	StatusEventTypeJobChanged        = "JOB_CHANGED"
	StatusEventTypeExperienceChanged = "EXPERIENCE_CHANGED"
	StatusEventTypeLevelChanged      = "LEVEL_CHANGED"
	StatusEventTypeMesoChanged       = "MESO_CHANGED"
	StatusEventTypeFameChanged       = "FAME_CHANGED"
	StatusEventTypeStatChanged       = "STAT_CHANGED"

	// ExperienceDistributionType* are the distribution kinds carried in an
	// EXPERIENCE_CHANGED event's Distributions slice. Each maps to a distinct
	// field of socket/model.IncreaseExperienceConfig, and each field renders a
	// DIFFERENT line in the client's experience-gain message.
	//
	// Only WHITE, YELLOW and CHAT set the primary "You have gained experience"
	// amount. EVERY other type is a secondary modifier line rendered IN
	// ADDITION to that primary line -- picking one of them alone renders
	// "You have gained experience (+0)" with a bonus line beneath it.
	//
	//	WHITE          Amount, White=true          "You have gained experience", white text
	//	YELLOW         Amount, White=false         "You have gained experience", yellow text
	//	CHAT           Amount, InChat=true         chat-window experience line
	//	MONSTER_BOOK   MonsterBookBonus            right side, yellow: Bonus Event EXP
	//	MONSTER_EVENT  MobEventBonusPercentage     in chat, pink: bonus EXP per 3rd monster defeated
	//	PLAY_TIME      MobEventBonusPercentage,    right side, yellow: Bonus EXP for hunting over
	//	               PlayTimeHour (from Attr1)   (N) hrs
	//	WEDDING        WeddingBonusEXP             right side, yellow: Bonus Wedding EXP
	//	SPIRIT_WEEK    QuestBonusRate              Earned 'Spirit Week Event' bonus EXP
	//	PARTY          PartyBonusExp,              right side, yellow: Bonus Event Party EXP
	//	               PartyBonusEventRate (Attr1)
	//	ITEM           ItemBonusEXP                right side, yellow: Equip Item Bonus EXP
	//	INTERNET_CAFE  PremiumIPExp                right side, yellow: Internet Cafe EXP Bonus
	//	RAINBOW_WEEK   RainbowWeekEventEXP         right side, yellow: Rainbow Week Bonus Event EXP
	//	PARTY_RING     PartyEXPRingEXP             v95+ only
	//	CAKE_PIE       CakePieEventBonus           v95+ only
	//
	// ITEM is the trap: it is the "Equip Item Bonus EXP" MODIFIER line, not
	// "experience that came from an item". See task-277,
	// docs/tasks/task-277-stored-exp-items/bug-redeem-renders-as-item-bonus.md.
	//
	// A new constant added here MUST also be added to
	// AllExperienceDistributionTypes below, or
	// TestExperienceDistributionTypeExhaustiveness will not notice it is
	// unmapped.
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

// AllExperienceDistributionTypes enumerates every ExperienceDistributionType*
// constant, in declaration order. Its sole purpose is exhaustiveness
// enforcement: TestExperienceDistributionTypeExhaustiveness iterates it and
// fails when a type has no case in the consumer's mapping table. Adding a
// constant above without adding it here silently defeats that check.
var AllExperienceDistributionTypes = []string{
	ExperienceDistributionTypeWhite,
	ExperienceDistributionTypeYellow,
	ExperienceDistributionTypeChat,
	ExperienceDistributionTypeMonsterBook,
	ExperienceDistributionTypeMonsterEvent,
	ExperienceDistributionTypePlayTime,
	ExperienceDistributionTypeWedding,
	ExperienceDistributionTypeSpiritWeek,
	ExperienceDistributionTypeParty,
	ExperienceDistributionTypeItem,
	ExperienceDistributionTypeInternetCafe,
	ExperienceDistributionTypeRainbowWeek,
	ExperienceDistributionTypePartyRing,
	ExperienceDistributionTypeCakePie,
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventStatChangedBody struct {
	ChannelId       channel.Id             `json:"channelId"`
	ExclRequestSent bool                   `json:"exclRequestSent"`
	Updates         []stat.Type            `json:"updates"`
	Values          map[string]interface{} `json:"values,omitempty"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
	// UseTargetPosition, when true, lands the character at the exact (TargetX,
	// TargetY) coordinate via the SET_FIELD chase mechanism instead of a named
	// portal — used by Mystic Door to place the user on the linked door.
	UseTargetPosition bool  `json:"useTargetPosition"`
	TargetX           int16 `json:"targetX"`
	TargetY           int16 `json:"targetY"`
}

// JobChangedStatusEventBody mirrors atlas-character's
// JobChangedStatusEventBody. It carries NO map id — which is why the dragon
// create for a job change has to run channel-side, where the session's field is
// available (see task-225 plan Task 9).
type JobChangedStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	JobId     job.Id     `json:"jobId"`
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
	Amount     int32 `json:"amount"`
	ShowEffect bool  `json:"showEffect"`
}
