package food

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// EnvCommandTopic is the channel -> consumables command topic for taming-mob
// (mount) food. The Command envelope below MUST match the channel producer
// (services/atlas-channel/.../kafka/message/food/kafka.go) field-for-field, or
// the JSON decode silently yields zero values.
const (
	EnvCommandTopic = "COMMAND_TOPIC_TAMING_MOB_FOOD"

	CommandRequestFeed = "REQUEST_FEED"
)

// Command mirrors the consumables command envelope
// (kafka/message/consumable/kafka.go). worldId flows through so the emitted
// TamingMobFed event (Task 33 -> atlas-mounts Task 20) can carry it.
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

// RequestFeedBody is the body for a taming-mob food consume request. The
// classification-226 gate is validated in the consumables consumer (Task 32).
type RequestFeedBody struct {
	Slot   int16  `json:"slot"`
	ItemId uint32 `json:"itemId"`
}

// EnvEventTopic carries the taming-mob food (feed) events produced by
// atlas-consumables (Task 33) and consumed by atlas-mounts (Task 20). The
// producer MUST populate worldId, characterId, itemId, and tirednessHeal.
const EnvEventTopic = "EVENT_TOPIC_TAMING_MOB_FOOD"

// Event is the taming-mob food event emitted after a successful revitalizer
// consume. This struct is the cross-service contract — it must match
// services/atlas-mounts/.../kafka/message/food/kafka.go field names and json
// tags exactly. worldId is sourced from the command envelope's WorldId.
type Event struct {
	WorldId       world.Id `json:"worldId"`
	CharacterId   uint32   `json:"characterId"`
	ItemId        uint32   `json:"itemId"`
	TirednessHeal int32    `json:"tirednessHeal"`
}

// RevitalizerTirednessHeal is the server-side pinned tiredness heal applied per
// revitalizer (classification 226). Task 8 confirmed the revitalizer heal is
// NOT WZ-data-driven (no spec field exists), so it is a fixed server constant.
const RevitalizerTirednessHeal = 30
