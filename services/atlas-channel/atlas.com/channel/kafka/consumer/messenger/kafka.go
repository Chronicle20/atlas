package messenger

const (
	EnvEventStatusTopic             = "EVENT_TOPIC_MESSENGER_STATUS"
	EventMessengerStatusTypeCreated = "CREATED"
	EventMessengerStatusTypeJoined  = "JOINED"
	EventMessengerStatusTypeLeft    = "LEFT"
	EventMessengerStatusTypeError   = "ERROR"
)

type statusEvent[E any] struct {
	ActorId     uint32 `json:"actorId"`
	WorldId     byte   `json:"worldId"`
	MessengerId uint32 `json:"messengerId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type createdEventBody struct {
}

type joinedEventBody struct {
	Slot byte `json:"slot"`
}

type leftEventBody struct {
	Slot byte `json:"slot"`
}

type errorEventBody struct {
	Type          string `json:"type"`
	CharacterName string `json:"characterName"`
}
