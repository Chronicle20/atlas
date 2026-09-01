package saga

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic     topic.Token = "COMMAND_TOPIC_SAGA"
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_SAGA_STATUS"
)

const (
	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

type StatusEventCompletedBody struct{}

type StatusEventFailedBody struct {
	ErrorCode  string `json:"errorCode"`
	Reason     string `json:"reason"`
	FailedStep string `json:"failedStep"`
}
