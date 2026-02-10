package account

import (
	"time"

	"github.com/google/uuid"
)

const (
	EnvEventSessionStatusTopic = "EVENT_TOPIC_ACCOUNT_SESSION_STATUS"

	SessionEventStatusTypeCreated      = "CREATED"
	SessionEventStatusTypeStateChanged = "STATE_CHANGED"
	SessionEventStatusTypeError        = "ERROR"
)

type SessionStatusEvent[E any] struct {
	SessionId   uuid.UUID `json:"sessionId"`
	AccountId   uint32    `json:"accountId"`
	AccountName string    `json:"accountName"`
	Type        string    `json:"type"`
	Body        E         `json:"body"`
}

type CreatedSessionStatusEventBody struct {
	IPAddress string `json:"ipAddress"`
	HWID      string `json:"hwid"`
}

type ErrorSessionStatusEventBody struct {
	Code      string `json:"code"`
	Reason    byte   `json:"reason"`
	Until     time.Time `json:"until"`
	IPAddress string `json:"ipAddress"`
	HWID      string `json:"hwid"`
}
