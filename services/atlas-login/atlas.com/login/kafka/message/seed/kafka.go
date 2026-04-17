package seed

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_SEED_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeFailed  = "FAILED"
)

type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type CreatedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

// FailedStatusEventBody mirrors the factory's shape (see
// atlas-character-factory/kafka/message/seed/kafka.go). Reason is optional —
// login uses it for log correlation but does not surface it to the client.
type FailedStatusEventBody struct {
	Reason string `json:"reason,omitempty"`
}
