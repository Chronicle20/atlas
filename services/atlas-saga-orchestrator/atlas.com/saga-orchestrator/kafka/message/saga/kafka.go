package saga

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_SAGA"
)

const (
	EnvStatusEventTopic      = "EVENT_TOPIC_SAGA_STATUS"
	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

// Error codes for saga failure events
const (
	ErrorCodeNotEnoughMesos = "NOT_ENOUGH_MESOS"
	ErrorCodeInventoryFull  = "INVENTORY_FULL"
	ErrorCodeStorageFull    = "STORAGE_FULL"
	ErrorCodeSagaTimeout    = "SAGA_TIMEOUT"
	ErrorCodeUnknown        = "UNKNOWN"
)

// MtsFailureKind* discriminate which MTS operation an mts_operation saga was
// performing when it failed, so atlas-channel can write the matching clientbound
// *Failed arm (buy -> BuyItemFailed, list -> RegisterSaleEntryFailed, take-home
// -> MoveItcPurchaseItemLtoSFailed) to unhang the originating dialog. Carried on
// StatusEventFailedBody.MtsKind. "mts_take_home" matches MtsTakeHomeResultKind so
// the take-home success/failure kinds line up.
const (
	MtsFailureKindBuy      = "mts_buy"
	MtsFailureKindList     = "mts_list"
	MtsFailureKindTakeHome = "mts_take_home"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventCompletedBody struct {
	SagaType    string         `json:"sagaType,omitempty"`
	Results     map[string]any `json:"results,omitempty"`
}

type StatusEventFailedBody struct {
	Reason      string `json:"reason"`
	FailedStep  string `json:"failedStep"`
	CharacterId uint32 `json:"characterId"`
	AccountId   uint32 `json:"accountId"`
	SagaType    string `json:"sagaType"`
	ErrorCode   string `json:"errorCode"`
	// MtsKind is set only for mts_operation sagas (one of MtsFailureKind*) so the
	// channel can route the failure to the correct MTS dialog arm. Empty otherwise.
	MtsKind string `json:"mtsKind,omitempty"`
}
