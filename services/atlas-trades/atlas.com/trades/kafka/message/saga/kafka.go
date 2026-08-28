// Package saga carries the COMMAND_TOPIC_SAGA / EVENT_TOPIC_SAGA_STATUS
// envelopes used to run a trade settlement. Mirrors
// services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/saga/kafka.go;
// struct names, field names and json tags must match that file exactly. Only
// the constants this service uses are carried over — the orchestrator's
// MTS-specific failure kinds are omitted, exactly as atlas-channel's mirror
// omits fields it does not read.
package saga

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_SAGA"
)

const (
	EnvStatusEventTopic      topic.Token = "EVENT_TOPIC_SAGA_STATUS"
	StatusEventTypeCompleted             = "COMPLETED"
	StatusEventTypeFailed                = "FAILED"
)

// Error codes for saga failure events
const (
	ErrorCodeNotEnoughMesos = "NOT_ENOUGH_MESOS"
	ErrorCodeInventoryFull  = "INVENTORY_FULL"
	ErrorCodeStorageFull    = "STORAGE_FULL"
	ErrorCodeSagaTimeout    = "SAGA_TIMEOUT"
	ErrorCodeUnknown        = "UNKNOWN"
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

// StatusEventFailedBody reports one failed step. CharacterId is the failed
// EXPANDED step's character — for a trade settlement that is whichever
// participant's release/accept step broke, never a role. Resolve both trade
// participants from TransactionId; never infer a side from CharacterId.
type StatusEventFailedBody struct {
	Reason      string `json:"reason"`
	FailedStep  string `json:"failedStep"`
	CharacterId uint32 `json:"characterId"`
	AccountId   uint32 `json:"accountId"`
	SagaType    string `json:"sagaType"`
	ErrorCode   string `json:"errorCode"`
	MtsKind     string `json:"mtsKind,omitempty"`
}
