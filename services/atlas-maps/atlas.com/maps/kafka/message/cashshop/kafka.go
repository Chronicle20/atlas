package cashshop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCashShopStatus           = "EVENT_TOPIC_CASH_SHOP_STATUS"
	EventCashShopStatusTypeCharacterEnter = "CHARACTER_ENTER"
	EventCashShopStatusTypeCharacterExit  = "CHARACTER_EXIT"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CharacterMovementBody struct {
	CharacterId uint32     `json:"characterId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
}