package consumable

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_CONSUMABLE"

	CommandRequestItemConsume   = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll        = "REQUEST_SCROLL"
	CommandRequestItemReward    = "REQUEST_ITEM_REWARD"
	CommandRequestVegaScroll    = "REQUEST_VEGA_SCROLL"
	CommandRequestViciousHammer = "REQUEST_VICIOUS_HAMMER"
	CommandRequestSkillBookUse  = "REQUEST_SKILL_BOOK_USE"
	CommandRequestCatchMonster  = "REQUEST_CATCH_MONSTER"
)

type Command[E any] struct {
	WorldId     world.Id     `json:"worldId"`
	ChannelId   channel.Id   `json:"channelId"`
	MapId       _map.Id      `json:"mapId"`
	Instance    uuid.UUID    `json:"instance"`
	CharacterId character.Id `json:"characterId"`
	Type        string       `json:"type"`
	Body        E            `json:"body"`
}

type RequestItemConsumeBody struct {
	Source   slot.Position `json:"source"`
	ItemId   item.Id       `json:"itemId"`
	Quantity int16         `json:"quantity"`
	PetId    uint64        `json:"petId,omitempty"`
}

type RequestItemRewardBody struct {
	Source slot.Position `json:"source"`
	ItemId item.Id       `json:"itemId"`
}

type RequestScrollBody struct {
	ScrollSlot      slot.Position `json:"scrollSlot"`
	EquipSlot       slot.Position `json:"equipSlot"`
	WhiteScroll     bool          `json:"whiteScroll"`
	LegendarySpirit bool          `json:"legendarySpirit"`
}

type RequestSkillBookUseBody struct {
	Slot   slot.Position `json:"slot"`
	ItemId item.Id       `json:"itemId"`
}

type RequestVegaScrollBody struct {
	VegaSlot   slot.Position `json:"vegaSlot"`
	VegaItemId item.Id       `json:"vegaItemId"`
	ScrollSlot slot.Position `json:"scrollSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}

type RequestViciousHammerBody struct {
	HammerSlot slot.Position `json:"hammerSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}

type RequestCatchMonsterBody struct {
	Source          slot.Position `json:"source"`
	ItemId          item.Id       `json:"itemId"`
	MonsterUniqueId uint32        `json:"monsterUniqueId"`
}

const (
	EnvEventTopic            topic.Token = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError                       = "ERROR"
	EventTypeScroll                      = "SCROLL"
	EventTypeSkillBookResult             = "SKILL_BOOK_RESULT"
	EventTypeVegaScroll                  = "VEGA_SCROLL"
	EventTypeViciousHammer               = "VICIOUS_HAMMER"

	EventTypeRewardEffect = "REWARD_EFFECT"
	EventTypeRewardWon    = "REWARD_WON"
	EventTypeCatchFailed  = "CATCH_FAILED"

	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
	ErrorTypeInventoryFull    = "INVENTORY_FULL"
	ErrorTypeVegaInvalid      = "VEGA_INVALID"

	// CatchCauseUseDelay / CatchCauseInventoryFull / CatchCauseInvalidItem are
	// the pre-reservation bridle-capture failure causes atlas-consumables
	// emits. The wire-reason mapping is resolved in atlas-channel (DOM-25) --
	// see bridleFailReason in kafka/consumer/monster/consumer.go.
	CatchCauseUseDelay      = "USE_DELAY"
	CatchCauseInventoryFull = "INVENTORY_FULL"
	CatchCauseInvalidItem   = "INVALID_ITEM"
)

type Event[E any] struct {
	CharacterId character.Id `json:"characterId"`
	Type        string       `json:"type"`
	Body        E            `json:"body"`
}

type ErrorBody struct {
	Error string `json:"error"`
}

type ScrollBody struct {
	Success         bool `json:"success"`
	Cursed          bool `json:"cursed"`
	LegendarySpirit bool `json:"legendarySpirit"`
	WhiteScroll     bool `json:"whiteScroll"`
}

type SkillBookResultBody struct {
	IsMasteryBook bool   `json:"isMasteryBook"`
	SkillId       uint32 `json:"skillId"`
	MasterLevel   uint32 `json:"masterLevel"`
	CanUse        bool   `json:"canUse"`
	Success       bool   `json:"success"`
}

type RewardEffectBody struct {
	BoxItemId uint32 `json:"boxItemId"`
	Effect    string `json:"effect"`
}

type RewardWonBody struct {
	BoxItemId uint32 `json:"boxItemId"`
	ItemId    uint32 `json:"itemId"`
	Message   string `json:"message"`
}

type VegaScrollBody struct {
	Success bool `json:"success"`
	Cursed  bool `json:"cursed"`
}

type ViciousHammerBody struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
}

// CatchFailedBody carries a bridle capture attempt rejected before the
// reservation reached atlas-monsters (delay gate, inventory full, invalid
// item). Cause is one of the CatchCause* constants above.
type CatchFailedBody struct {
	ItemId uint32 `json:"itemId"`
	Cause  string `json:"cause"`
}
