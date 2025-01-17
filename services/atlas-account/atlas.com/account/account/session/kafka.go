package session

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_ACCOUNT_SESSION"

	CommandIssuerInternal = "INTERNAL"

	CommandTypeLogout = "LOGOUT"

	EnvEventStatusTopic                    = "EVENT_TOPIC_ACCOUNT_SESSION_STATUS"
	EventStatusTypeCreated                 = "CREATED"
	EventStatusTypeStateChanged            = "STATE_CHANGED"
	EventStatusTypeRequestLicenseAgreement = "REQUEST_LICENSE_AGREEMENT"
	EventStatusTypeError                   = "ERROR"
)

type command[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Issuer    string    `json:"author"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type logoutCommandBody struct {
}

type statusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type createdStatusEventBody struct {
}

type stateChangedEventBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type errorStatusEventBody struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}
