package character

const (
	EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeDeleted       = "DELETED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventDeletedBody struct {
}
