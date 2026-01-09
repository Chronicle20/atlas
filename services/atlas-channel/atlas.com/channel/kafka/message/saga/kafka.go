package saga

import "github.com/google/uuid"

const (
	EnvCommandTopic     = "COMMAND_TOPIC_SAGA"
	EnvStatusEventTopic = "EVENT_TOPIC_SAGA_STATUS"

	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

type StatusEventCompletedBody struct {
}

type StatusEventFailedBody struct {
	Reason     string `json:"reason"`
	FailedStep string `json:"failedStep"`
}
