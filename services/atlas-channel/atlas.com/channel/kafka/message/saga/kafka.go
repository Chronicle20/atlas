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
	SagaTypePointReset       = "point_reset"
	SagaTypeMtsOperation     = "mts_operation"
)

// MtsTakeHomeResultKind is the Results["kind"] marker the orchestrator sets on a
// completed WithdrawFromMts (take-home) saga so this service can recognize it and
// write MoveItcPurchaseItemLtoSDone. Mirrors saga.MtsTakeHomeResultKind in
// atlas-saga-orchestrator.
const MtsTakeHomeResultKind = "mts_take_home"

// MtsFailureKind* mirror the orchestrator's MtsFailureKind* (kafka/message/saga
// in atlas-saga-orchestrator): they discriminate which MTS operation a failed
// mts_operation saga was performing so handleFailedEvent can write the matching
// clientbound *Failed arm to unhang the originating dialog.
const (
	MtsFailureKindBuy      = "mts_buy"
	MtsFailureKindList     = "mts_list"
	MtsFailureKindTakeHome = "mts_take_home"
)

type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

type StatusEventCompletedBody struct {
	SagaType string         `json:"sagaType,omitempty"`
	Results  map[string]any `json:"results,omitempty"`
}

type StatusEventFailedBody struct {
	Reason      string `json:"reason"`
	FailedStep  string `json:"failedStep"`
	CharacterId uint32 `json:"characterId"`
	SagaType    string `json:"sagaType"`
	ErrorCode   string `json:"errorCode"`
	// MtsKind is set only for mts_operation sagas (one of MtsFailureKind*).
	MtsKind string `json:"mtsKind,omitempty"`
}
