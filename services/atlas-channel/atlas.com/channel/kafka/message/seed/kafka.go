package seed

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_SEED_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeFailed  = "FAILED"
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
// Reason is optional; consumers may log it but must not surface it verbatim
// to the client.
type FailedStatusEventBody struct {
	Reason string `json:"reason,omitempty"`
}
