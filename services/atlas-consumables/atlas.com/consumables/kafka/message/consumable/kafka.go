package consumable

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandRequestItemConsume     = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll          = "REQUEST_SCROLL"
	CommandRequestVegaScroll      = "REQUEST_VEGA_SCROLL"
	CommandRequestSkillBookUse    = "REQUEST_SKILL_BOOK_USE"
	CommandApplyConsumableEffect  = "APPLY_CONSUMABLE_EFFECT"
	CommandCancelConsumableEffect = "CANCEL_CONSUMABLE_EFFECT"
	CommandRequestItemReward      = "REQUEST_ITEM_REWARD"
	CommandRequestViciousHammer   = "REQUEST_VICIOUS_HAMMER"
)

type Command[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	ChannelId     channel.Id   `json:"channelId"`
	MapId         _map.Id      `json:"mapId"`
	Instance      uuid.UUID    `json:"instance"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

type RequestItemConsumeBody struct {
	Source   slot.Position `json:"source"`
	ItemId   item.Id       `json:"itemId"`
	Quantity int16         `json:"quantity"`
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

// RequestSkillBookUseBody is the body for a Skill Book (228) / Mastery Book
// (229) consume request (task-125). Slot is the USE-compartment slot holding
// the book.
type RequestSkillBookUseBody struct {
	Slot   slot.Position `json:"slot"`
	ItemId item.Id       `json:"itemId"`
}

// RequestVegaScrollBody asks the service to apply the scroll at ScrollSlot to
// the equip at EquipSlot at the vega-boosted rate, consuming the vega cash
// item at VegaSlot together with the scroll. EquipSlot sign convention:
// positive = equip inventory (the vega dialog's targets), negative = equipped.
type RequestVegaScrollBody struct {
	VegaSlot   slot.Position `json:"vegaSlot"`   // cash compartment
	VegaItemId item.Id       `json:"vegaItemId"` // re-validated against slot contents
	ScrollSlot slot.Position `json:"scrollSlot"` // use compartment
	EquipSlot  slot.Position `json:"equipSlot"`
}

// RequestViciousHammerBody carries the two slots the CUIItemUpgrade dialog
// round-trip token packs: the hammer's cash-compartment slot and the target
// equip slot (negative = equipped, positive = equip inventory).
type RequestViciousHammerBody struct {
	HammerSlot slot.Position `json:"hammerSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}

// ApplyConsumableEffectBody is the body for applying consumable effects without consuming from inventory
// Used for NPC-initiated buffs (e.g., NPC blessings)
type ApplyConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}

// CancelConsumableEffectBody is the body for cancelling consumable effects on a character
// Used for portal-initiated buff cancellation (e.g., removing draco buff after transit)
type CancelConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}

const (
	EnvEventTopic            = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError           = "ERROR"
	EventTypeScroll          = "SCROLL"
	EventTypeVegaScroll      = "VEGA_SCROLL"
	EventTypeEffectApplied   = "EFFECT_APPLIED"
	EventTypeRewardEffect    = "REWARD_EFFECT"
	EventTypeRewardWon       = "REWARD_WON"
	EventTypeViciousHammer   = "VICIOUS_HAMMER"
	EventTypeSkillBookResult = "SKILL_BOOK_RESULT"

	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
	ErrorTypeInventoryFull    = "INVENTORY_FULL"
	ErrorTypeVegaInvalid      = "VEGA_INVALID"
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

// SkillBookResultBody carries the outcome of a skill-book use — the writer's
// inputs for SKILL_LEARN_ITEM_RESULT. CanUse=false is a validation rejection
// or saga failure (requester-only); CanUse=true broadcasts to the map with
// Success carrying the roll outcome.
type SkillBookResultBody struct {
	IsMasteryBook bool   `json:"isMasteryBook"`
	SkillId       uint32 `json:"skillId"`
	MasterLevel   uint32 `json:"masterLevel"`
	CanUse        bool   `json:"canUse"`
	Success       bool   `json:"success"`
}

// VegaScrollBody carries the resolved vega scroll outcome. Distinct from
// ScrollBody so the channel can emit the CUIVega dialog packets instead of
// the plain map broadcast; whiteScroll/legendarySpirit are always false on
// the vega path and therefore not carried.
type VegaScrollBody struct {
	Success bool `json:"success"`
	Cursed  bool `json:"cursed"`
}

// ViciousHammerBody reports the terminal result of a hammer use. Reason is the
// SEMANTIC failure notice (NOT_UPGRADABLE / CAP_REACHED / HORNTAIL / UNKNOWN);
// atlas-channel resolves it to the client wire byte per tenant (DOM-25).
// Meaningful when !Success.
type ViciousHammerBody struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
}

type EffectAppliedBody struct {
	ItemId        item.Id   `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
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
