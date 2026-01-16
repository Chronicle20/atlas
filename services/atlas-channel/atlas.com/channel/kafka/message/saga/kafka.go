package saga

import "github.com/google/uuid"

const (
	EnvCommandTopic     = "COMMAND_TOPIC_SAGA"
	EnvStatusEventTopic = "EVENT_TOPIC_SAGA_STATUS"

	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

// Error codes for saga failure events
const (
	ErrorCodeNotEnoughMesos = "NOT_ENOUGH_MESOS"
	ErrorCodeInventoryFull  = "INVENTORY_FULL"
	ErrorCodeStorageFull    = "STORAGE_FULL"
	ErrorCodeUnknown        = "UNKNOWN"
)

// Saga type constants
const (
	SagaTypeStorageOperation = "storage_operation"
)

type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

type StatusEventCompletedBody struct {
}

type StatusEventFailedBody struct {
	Reason      string `json:"reason"`
	FailedStep  string `json:"failedStep"`
	CharacterId uint32 `json:"characterId"`
	SagaType    string `json:"sagaType"`
	ErrorCode   string `json:"errorCode"`
}
