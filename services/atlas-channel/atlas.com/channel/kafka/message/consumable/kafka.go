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

	CommandRequestItemConsume = "REQUEST_ITEM_CONSUME"
	CommandRequestScroll      = "REQUEST_SCROLL"
	CommandRequestVegaScroll  = "REQUEST_VEGA_SCROLL"
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
}

type RequestScrollBody struct {
	ScrollSlot      slot.Position `json:"scrollSlot"`
	EquipSlot       slot.Position `json:"equipSlot"`
	WhiteScroll     bool          `json:"whiteScroll"`
	LegendarySpirit bool          `json:"legendarySpirit"`
}

type RequestVegaScrollBody struct {
	VegaSlot   slot.Position `json:"vegaSlot"`
	VegaItemId item.Id       `json:"vegaItemId"`
	ScrollSlot slot.Position `json:"scrollSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}

const (
	EnvEventTopic       = "EVENT_TOPIC_CONSUMABLE_STATUS"
	EventTypeError      = "ERROR"
	EventTypeScroll     = "SCROLL"
	EventTypeVegaScroll = "VEGA_SCROLL"

	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
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

type VegaScrollBody struct {
	Success bool `json:"success"`
	Cursed  bool `json:"cursed"`
}
