package session

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_ACCOUNT_SESSION"

	CommandIssuerLogin = "LOGIN"

	CommandTypeCreate        = "CREATE"
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

type CreateCommandBody struct {
	AccountName string `json:"accountName"`
	Password    string `json:"password"`
	IPAddress   string `json:"ipAddress"`
}

type ProgressStateCommandBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type LogoutCommandBody struct {
}

const (
	EnvEventStatusTopic = "EVENT_TOPIC_ACCOUNT_SESSION_STATUS"

	EventStatusTypeCreated                 = "CREATED"
	EventStatusTypeStateChanged            = "STATE_CHANGED"
	EventStatusTypeRequestLicenseAgreement = "REQUEST_LICENSE_AGREEMENT"
	EventStatusTypeError                   = "ERROR"

	EventStatusErrorCodeSystemError       = "SYSTEM_ERROR"
	EventStatusErrorCodeNotRegistered     = "NOT_REGISTERED"
	EventStatusErrorCodeDeletedOrBlocked  = "DELETED_OR_BLOCKED"
	EventStatusErrorCodeAlreadyLoggedIn   = "ALREADY_LOGGED_IN"
	EventStatusErrorCodeIncorrectPassword = "INCORRECT_PASSWORD"
	EventStatusErrorCodeTooManyAttempts   = "TOO_MANY_ATTEMPTS"
)

type StatusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

type StateChangedEventBody[E any] struct {
	State  uint8 `json:"state"`
	Params E     `json:"params"`
}

type ErrorStatusEventBody struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}
