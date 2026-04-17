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

// FailedStatusEventBody carries log-correlation detail for a seed failure.
// Reason is optional; login does not use it for client messaging (see PRD §4.5 /
// plan Phase 8) but consumers may log it.
type FailedStatusEventBody struct {
	Reason string `json:"reason,omitempty"`
}
