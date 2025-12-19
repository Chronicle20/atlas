package character

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreatedStatusEventBody struct {
	Name string `json:"name"`
}

type DeletedStatusEventBody struct {
}
