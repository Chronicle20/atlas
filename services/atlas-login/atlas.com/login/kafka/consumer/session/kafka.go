package session

import (
	"github.com/google/uuid"
)

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

type statusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type createdStatusEventBody struct {
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
