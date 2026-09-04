package _map

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/backeffect"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopicMap topic.Token = "COMMAND_TOPIC_MAP"
)

const (
	CommandTypeWeatherStart    = "WEATHER_START"
	CommandTypeSetBackEffect   = "SET_BACK_EFFECT"
	CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"
)

type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type WeatherStartCommandBody struct {
	ItemId     uint32 `json:"itemId"`
	Message    string `json:"message"`
	DurationMs uint32 `json:"durationMs"`
}

type SetBackEffectCommandBody struct {
	Effect   backeffect.Effect `json:"effect"`
	FieldId  uint32            `json:"fieldId"`
	PageId   uint8             `json:"pageId"`
	Duration uint32            `json:"duration"`
}

type ClearBackEffectCommandBody struct{}
