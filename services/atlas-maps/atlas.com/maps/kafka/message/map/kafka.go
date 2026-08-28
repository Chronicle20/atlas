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
	EventTopicMapStatusTypeBackEffectSet   = "BACK_EFFECT_SET"
	EventTopicMapStatusTypeBackEffectClear = "BACK_EFFECT_CLEAR"

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

// EnvironmentObject is one object that was tracked and cleared by a reset.
type EnvironmentObject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// EnvironmentReset carries every entry that was tracked, i.e. which objects
// were cleared. The channel decides what, if anything, to announce for each:
// obstacles are restored via FieldObstacleAllReset plus an explicit state-0
// announce; no state hides a named (non-obstacle) object, so those are simply
// cleared with no announce -- see design.md section 1.2.
type EnvironmentReset struct {
	Cleared []EnvironmentObject `json:"cleared"`
}

type BackEffectSet struct {
	Effect   backeffect.Effect `json:"effect"`
	FieldId  uint32            `json:"fieldId"`
	PageId   uint8             `json:"pageId"`
	Duration uint32            `json:"duration"`
}

type BackEffectClear struct{}
