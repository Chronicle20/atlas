package saga

import "github.com/google/uuid"

// EnvCommandTopic is the saga command topic env var name
// (deploy/k8s/base/env-configmap.yaml), shared across every
// saga-emitting service.
//
// EnvStatusEventTopic is the saga status event topic env var name every
// saga participant consumes to learn a saga it emitted reached a terminal
// state.
const (
	EnvCommandTopic     = "COMMAND_TOPIC_SAGA"
	EnvStatusEventTopic = "EVENT_TOPIC_SAGA_STATUS"
)

// StatusEventTypeCompleted and StatusEventTypeFailed are StatusEvent.Type's
// only two terminal values (mirrors
// services/atlas-map-actions/atlas.com/map-actions/kafka/message/saga/kafka.go).
const (
	StatusEventTypeCompleted = "COMPLETED"
	StatusEventTypeFailed    = "FAILED"
)

// StatusEvent is the saga orchestrator's terminal-state event, keyed by the
// saga's transaction id -- the only handle it carries back to whichever
// service emitted the saga.
type StatusEvent[T any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          T         `json:"body"`
}

// StatusEventCompletedBody is StatusEvent's body when Type is
// StatusEventTypeCompleted.
type StatusEventCompletedBody struct{}

// StatusEventFailedBody is StatusEvent's body when Type is
// StatusEventTypeFailed.
type StatusEventFailedBody struct {
	ErrorCode  string `json:"errorCode"`
	Reason     string `json:"reason"`
	FailedStep string `json:"failedStep"`
}
