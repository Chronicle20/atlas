package messenger

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_MESSENGER"
)

const (
	CommandMessengerCreate        = "CREATE"
	CommandMessengerLeave         = "LEAVE"
	CommandMessengerRequestInvite = "REQUEST_INVITE"
)

type Command[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type CreateCommandBody struct{}

type LeaveCommandBody struct {
	MessengerId uint32 `json:"messengerId"`
}

type RequestInviteBody struct {
	CharacterId uint32 `json:"characterId"`
}

const (
	EnvEventStatusTopic topic.Token = "EVENT_TOPIC_MESSENGER_STATUS"
)

const (
	EventMessengerStatusTypeCreated = "CREATED"
	EventMessengerStatusTypeJoined  = "JOINED"
	EventMessengerStatusTypeLeft    = "LEFT"
	EventMessengerStatusTypeError   = "ERROR"
)

type StatusEvent[E any] struct {
	ActorId     uint32   `json:"actorId"`
	WorldId     world.Id `json:"worldId"`
	MessengerId uint32   `json:"messengerId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type CreatedEventBody struct{}

type JoinedEventBody struct {
	Slot byte `json:"slot"`
}

type LeftEventBody struct {
	Slot byte `json:"slot"`
}

type ErrorEventBody struct {
	Type          string `json:"type"`
	CharacterName string `json:"characterName"`
}
