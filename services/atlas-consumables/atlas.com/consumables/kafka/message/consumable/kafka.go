package consumable

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandRequestItemConsume     = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll          = "REQUEST_SCROLL"
	CommandApplyConsumableEffect  = "APPLY_CONSUMABLE_EFFECT"
	CommandCancelConsumableEffect = "CANCEL_CONSUMABLE_EFFECT"
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

type RequestScrollBody struct {
	ScrollSlot      slot.Position `json:"scrollSlot"`
	EquipSlot       slot.Position `json:"equipSlot"`
	WhiteScroll     bool          `json:"whiteScroll"`
	LegendarySpirit bool          `json:"legendarySpirit"`
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
	EnvEventTopic          = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError         = "ERROR"
	EventTypeScroll        = "SCROLL"
	EventTypeEffectApplied = "EFFECT_APPLIED"
	EventTypeViciousHammer = "VICIOUS_HAMMER"

	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
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
