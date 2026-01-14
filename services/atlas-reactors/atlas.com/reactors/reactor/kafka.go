package reactor

import "time"

const (
	EnvCommandTopic   = "COMMAND_TOPIC_REACTOR"
	CommandTypeCreate = "CREATE"
)

type Command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type CreateCommandBody struct {
	Classification uint32 `json:"classification"`
	Name           string `json:"name"`
	State          int8   `json:"state"`
	X              int16  `json:"x"`
	Y              int16  `json:"y"`
	Delay          uint32 `json:"delay"`
	Direction      byte   `json:"direction"`
}

const (
	EnvEventStatusTopic      = "EVENT_TOPIC_REACTOR_STATUS"
	EventStatusTypeCreated   = "CREATED"
	EventStatusTypeDestroyed = "DESTROYED"
)

type statusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	ReactorId uint32 `json:"reactorId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type createdStatusEventBody struct {
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

type destroyedStatusEventBody struct {
	State int8  `json:"state"`
	X     int16 `json:"x"`
	Y     int16 `json:"y"`
}
