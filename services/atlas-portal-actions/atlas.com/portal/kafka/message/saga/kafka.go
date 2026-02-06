package saga

import "github.com/google/uuid"

const (
	EnvCommandTopic     = "COMMAND_TOPIC_SAGA"
	EnvStatusEventTopic = "EVENT_TOPIC_SAGA_STATUS"
)

// Status event types
const (
	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

// StatusEvent represents a saga status event from the orchestrator
type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

// StatusEventCompletedBody represents the body of a completed status event
type StatusEventCompletedBody struct{}

// StatusEventFailedBody represents the body of a failed status event
type StatusEventFailedBody struct {
	ErrorCode  string `json:"errorCode"`
	Reason     string `json:"reason"`
	FailedStep string `json:"failedStep"`
}
