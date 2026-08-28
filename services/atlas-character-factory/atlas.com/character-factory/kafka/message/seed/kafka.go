package seed

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicStatus    topic.Token = "EVENT_TOPIC_SEED_STATUS"
	StatusEventTypeCreated             = "CREATED"
	StatusEventTypeFailed              = "FAILED"
)

type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	// TransactionId correlates this event with the POST characters/seed
	// response that started it. Optional: an older producer omits it and
	// consumers fall back to AccountId (task-246 design §4.3).
	TransactionId string `json:"transactionId,omitempty"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
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
