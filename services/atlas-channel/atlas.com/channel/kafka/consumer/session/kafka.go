package session

import (
	"github.com/google/uuid"
)

const (
	EnvEventStatusTopic = "EVENT_TOPIC_ACCOUNT_SESSION_STATUS"

	EventStatusTypeStateChanged = "STATE_CHANGED"
	EventStatusTypeError        = "ERROR"

	EventStatusErrorCodeSystemError = "SYSTEM_ERROR"
)

type statusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type stateChangedEventBody[E any] struct {
	State  uint8 `json:"state"`
	Params E     `json:"params"`
}

type errorStatusEventBody struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}
