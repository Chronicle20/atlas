package reactor

import "time"

const (
	EnvEventStatusTopic      = "EVENT_TOPIC_REACTOR_STATUS"
	EventStatusTypeCreated   = "CREATED"
	EventStatusTypeDestroyed = "DESTROYED"
)

type StatusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	ReactorId uint32 `json:"reactorId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type CreatedStatusEventBody struct {
	Classification uint32    `json:"classification"`
	Name           string    `json:"name"`
	State          int8      `json:"state"`
	EventState     byte      `json:"eventState"`
	Delay          uint32    `json:"delay"`
	Direction      byte      `json:"direction"`
	X              int16     `json:"x"`
	Y              int16     `json:"y"`
	UpdateTime     time.Time `json:"updateTime"`
}

type DestroyedStatusEventBody struct {
	State int8  `json:"state"`
	X     int16 `json:"x"`
	Y     int16 `json:"y"`
}
