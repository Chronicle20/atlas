package saga

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_SAGA"
)

const (
	EnvStatusEventTopic       = "EVENT_TOPIC_SAGA_STATUS"
	StatusEventTypeCompleted  = "COMPLETED"
)

type StatusEvent[E any] struct {
	SagaId        uuid.UUID `json:"sagaId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

type StatusEventCompletedBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
}
