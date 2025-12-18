package inventory

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_INVENTORY_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreatedStatusEventBody struct {
}

type DeletedStatusEventBody struct {
}
