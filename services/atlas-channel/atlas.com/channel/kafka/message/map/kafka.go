package _map

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicMapStatus topic.Token = "EVENT_TOPIC_MAP_STATUS"
)

const (
	EventTopicMapStatusTypeCharacterEnter  = "CHARACTER_ENTER"
	EventTopicMapStatusTypeCharacterExit   = "CHARACTER_EXIT"
	EventTopicMapStatusTypeWeatherStart    = "WEATHER_START"
	EventTopicMapStatusTypeWeatherEnd      = "WEATHER_END"
	EventTopicMapStatusTypeMapTimerStarted = "MAP_TIMER_STARTED"
	EventTopicMapStatusTypeJukeboxStart    = "JUKEBOX_START"
	EventTopicMapStatusTypeJukeboxEnd      = "JUKEBOX_END"

	EventTopicMapStatusTypeEnvironmentStateChanged = "ENVIRONMENT_STATE_CHANGED"
	EventTopicMapStatusTypeEnvironmentReset        = "ENVIRONMENT_RESET"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type CharacterEnter struct {
	CharacterId uint32 `json:"characterId"`
}

type CharacterExit struct {
	CharacterId uint32 `json:"characterId"`
}

type WeatherStart struct {
	ItemId  uint32 `json:"itemId"`
	Message string `json:"message"`
}

type WeatherEnd struct {
	ItemId uint32 `json:"itemId"`
}

type MapTimerStarted struct {
	CharacterId uint32 `json:"characterId"`
	Seconds     uint32 `json:"seconds"`
}

type JukeboxStart struct {
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}

type JukeboxEnd struct {
	ItemId uint32 `json:"itemId"`
}

type EnvironmentStateChanged struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

// EnvironmentObject is one cleared object. State is the state the reset must
// restore the object to -- the default the map declares for it, not the state
// it was cleared from.
type EnvironmentObject struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

// EnvironmentReset carries every entry that was tracked. It is not an empty
// body: FieldObstacleAllReset restores only the client's obstacle list, so the
// channel must be told which non-obstacle objects to restore, and to what
// state, and the channel keeps no registry of its own -- see design.md
// section 1.2.
type EnvironmentReset struct {
	Cleared []EnvironmentObject `json:"cleared"`
}
