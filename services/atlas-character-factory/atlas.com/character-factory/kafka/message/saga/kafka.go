package saga

import "github.com/google/uuid"

const (
	EnvCommandTopic = "COMMAND_TOPIC_SAGA"
)

const (
	EnvStatusEventTopic      = "EVENT_TOPIC_SAGA_STATUS"
	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventCompletedBody struct {
	SagaType string         `json:"sagaType,omitempty"`
	Results  map[string]any `json:"results,omitempty"`
}

// StatusEventFailedBody mirrors the orchestrator's body (see
// atlas-saga-orchestrator/kafka/message/saga/kafka.go). The factory needs
// `AccountId` (Phase 1.1) to re-emit a login-addressable FAILED event toward
// atlas-login, and `SagaType` to filter for CharacterCreation failures.
type StatusEventFailedBody struct {
	Reason      string `json:"reason"`
	FailedStep  string `json:"failedStep"`
	CharacterId uint32 `json:"characterId"`
	AccountId   uint32 `json:"accountId"`
	SagaType    string `json:"sagaType"`
	ErrorCode   string `json:"errorCode"`
}
