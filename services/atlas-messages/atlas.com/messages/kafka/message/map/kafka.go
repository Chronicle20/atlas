package _map

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopicMap      = "COMMAND_TOPIC_MAP"
	CommandTypeWeatherStart = "WEATHER_START"
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
