package saga

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

// Condition type constants for character state validation
const (
	JobCondition                    = "jobId"
	MesoCondition                   = "meso"
	MapCondition                    = "mapId"
	FameCondition                   = "fame"
	ItemCondition                   = "item"
	GenderCondition                 = "gender"
	LevelCondition                  = "level"
	RebornsCondition                = "reborns"
	DojoPointsCondition             = "dojoPoints"
	VanquisherKillsCondition        = "vanquisherKills"
	GmLevelCondition                = "gmLevel"
	GuildIdCondition                = "guildId"
	GuildLeaderCondition            = "guildLeader"
	GuildRankCondition              = "guildRank"
	QuestStatusCondition            = "questStatus"
	QuestProgressCondition          = "questProgress"
	UnclaimedMarriageGiftsCondition = "hasUnclaimedMarriageGifts"
	StrengthCondition               = "strength"
	DexterityCondition              = "dexterity"
	IntelligenceCondition           = "intelligence"
	LuckCondition                   = "luck"
	BuddyCapacityCondition          = "buddyCapacity"
	PetCountCondition               = "petCount"
	MapCapacityCondition            = "mapCapacity"
	InventorySpaceCondition         = "inventorySpace"
	TransportAvailableCondition     = "transportAvailable"
	SkillLevelCondition             = "skillLevel"
	HpCondition                     = "hp"
	MaxHpCondition                  = "maxHp"
	BuffCondition                   = "buff"
	ExcessSPCondition               = "excessSp"
	PartyIdCondition                = "partyId"
	PartyLeaderCondition            = "partyLeader"
	PartySizeCondition              = "partySize"
	PqCustomDataCondition           = "pqCustomData"
)

// Operator constants for validation conditions
const (
	Equals       = "="
	GreaterThan  = ">"
	LessThan     = "<"
	GreaterEqual = ">="
	LessEqual    = "<="
	In           = "in"
)

// ValidationConditionInput represents a condition for character state validation.
// This is the canonical wire format for condition inputs sent between services.
type ValidationConditionInput struct {
	Type            string     `json:"type"`
	Operator        string     `json:"operator"`
	Value           int        `json:"value"`
	Values          []int      `json:"values,omitempty"`
	ReferenceId     uint32     `json:"referenceId,omitempty"`
	Step            string     `json:"step,omitempty"`
	WorldId         world.Id   `json:"worldId,omitempty"`
	ChannelId       channel.Id `json:"channelId,omitempty"`
	IncludeEquipped bool       `json:"includeEquipped,omitempty"`
}
