package messenger

import (
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic               = "COMMAND_TOPIC_MESSENGER"
	CommandMessengerCreate        = "CREATE"
	CommandMessengerLeave         = "LEAVE"
	CommandMessengerRequestInvite = "REQUEST_INVITE"
)

type Command[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type CreateCommandBody struct {
}

type LeaveCommandBody struct {
	MessengerId uint32 `json:"messengerId"`
}

type RequestInviteBody struct {
	CharacterId uint32 `json:"characterId"`
}

const (
	EnvEventStatusTopic             = "EVENT_TOPIC_MESSENGER_STATUS"
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

type CreatedEventBody struct {
}

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
