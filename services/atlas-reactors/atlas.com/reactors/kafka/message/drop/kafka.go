package drop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicDropStatus = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeCreated  = "CREATED"
)

const (
	EnvCommandTopicDrop = "COMMAND_TOPIC_DROP"
	CommandTypeConsume   = "CONSUME"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	DropId        uint32     `json:"dropId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type StatusEventCreatedBody struct {
	ItemId          uint32 `json:"itemId"`
	Quantity        uint32 `json:"quantity"`
	X               int16  `json:"x"`
	Y               int16  `json:"y"`
	OwnerId         uint32 `json:"ownerId"`
	PlayerDrop      bool   `json:"playerDrop"`
}

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CommandConsumeBody struct {
	DropId uint32 `json:"dropId"`
}
