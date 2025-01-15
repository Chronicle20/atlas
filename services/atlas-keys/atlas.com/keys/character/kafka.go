package character

const (
	EnvEventTopicCharacterStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated = "CREATED"
)

type statusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	WorldId     byte   `json:"worldId"`
	Body        E      `json:"body"`
}

type statusEventCreatedBody struct {
	Name string `json:"name"`
}
