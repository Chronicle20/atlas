package seed

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_COMPARTMENT_STATUS"
	StatusEventTypeCreated = "CREATED"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreatedStatusEventBody struct {
}
