package consumable

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandRequestItemConsume    = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll         = "REQUEST_SCROLL"
	CommandApplyConsumableEffect = "APPLY_CONSUMABLE_EFFECT"
)

type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type RequestItemConsumeBody struct {
	Source   int16  `json:"source"`
	ItemId   uint32 `json:"itemId"`
	Quantity int16  `json:"quantity"`
}

type RequestScrollBody struct {
	ScrollSlot      int16 `json:"scrollSlot"`
	EquipSlot       int16 `json:"equipSlot"`
	WhiteScroll     bool  `json:"whiteScroll"`
	LegendarySpirit bool  `json:"legendarySpirit"`
}

// ApplyConsumableEffectBody is the body for applying consumable effects without consuming from inventory
// Used for NPC-initiated buffs (e.g., NPC blessings)
type ApplyConsumableEffectBody struct {
	ItemId uint32 `json:"itemId"`
}

const (
	EnvEventTopic   = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError  = "ERROR"
	EventTypeScroll = "SCROLL"
	EventTypeEffectApplied = "EFFECT_APPLIED"

	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
)

type Event[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
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

type EffectAppliedBody struct {
	ItemId        uint32    `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}
