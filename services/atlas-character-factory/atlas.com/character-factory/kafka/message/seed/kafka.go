package seed

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_SEED_STATUS"
	StatusEventTypeCreated = "CREATED"
)

type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type CreatedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
}
