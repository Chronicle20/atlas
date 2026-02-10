package session

import (
	"time"

	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_ACCOUNT_SESSION"

	CommandIssuerInternal = "INTERNAL"
	CommandIssuerLogin    = "LOGIN"
	CommandIssuerChannel  = "CHANNEL"

	CommandTypeProgressState = "PROGRESS_STATE"
	CommandTypeLogout        = "LOGOUT"
)

type Command[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Issuer    string    `json:"author"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type ProgressStateCommandBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type LogoutCommandBody struct {
}

const (
	EnvEventStatusTopic = "EVENT_TOPIC_ACCOUNT_SESSION_STATUS"

	EventStatusTypeStateChanged = "STATE_CHANGED"
	EventStatusTypeError        = "ERROR"

	EventStatusErrorCodeSystemError = "SYSTEM_ERROR"
)

type StatusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type StateChangedEventBody[E any] struct {
	State  uint8 `json:"state"`
	Params E     `json:"params"`
}

type ErrorStatusEventBody struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  time.Time `json:"until"`
}
