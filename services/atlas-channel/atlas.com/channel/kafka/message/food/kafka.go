package food

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EnvCommandTopic is the channel -> consumables command topic for taming-mob
// (mount) food. Task 32's consumables consumer MUST decode this Command shape.
const (
	EnvCommandTopic = "COMMAND_TOPIC_TAMING_MOB_FOOD"

	CommandRequestFeed = "REQUEST_FEED"
)

// Command mirrors the consumables command envelope
// (services/atlas-consumables/.../kafka/message/consumable/kafka.go). worldId
// flows to consumables via WorldId/ChannelId/MapId/Instance so the eventual
// fed event (Task 33 -> 20) can carry it. Field names and json tags MUST match
// the consumables consumer exactly or decode silently yields zero values.
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
// classification-226 gate is validated downstream in consumables (Task 32).
type RequestFeedBody struct {
	Slot   int16  `json:"slot"`
	ItemId uint32 `json:"itemId"`
}
