package account

import "github.com/google/uuid"

const (
	EnvCommandTopicCreateAccount = "COMMAND_TOPIC_CREATE_ACCOUNT"

	EnvCommandSessionTopic = "COMMAND_TOPIC_ACCOUNT_SESSION"

	SessionCommandIssuerInternal = "INTERNAL"
	SessionCommandIssuerLogin    = "LOGIN"
	SessionCommandIssuerChannel  = "CHANNEL"

	SessionCommandTypeCreate        = "CREATE"
	SessionCommandTypeProgressState = "PROGRESS_STATE"
	SessionCommandTypeLogout        = "LOGOUT"
)

type CreateCommand struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type SessionCommand[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Issuer    string    `json:"author"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type CreateSessionCommandBody struct {
	AccountName string `json:"accountName"`
	Password    string `json:"password"`
	IPAddress   string `json:"ipAddress"`
}

type ProgressStateSessionCommandBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type LogoutSessionCommandBody struct {
}

const (
	EnvEventTopicStatus  = "EVENT_TOPIC_ACCOUNT_STATUS"
	EventStatusCreated   = "CREATED"
	EventStatusLoggedIn  = "LOGGED_IN"
	EventStatusLoggedOut = "LOGGED_OUT"

	EnvEventSessionStatusTopic                    = "EVENT_TOPIC_ACCOUNT_SESSION_STATUS"
	SessionEventStatusTypeCreated                 = "CREATED"
	SessionEventStatusTypeStateChanged            = "STATE_CHANGED"
	SessionEventStatusTypeRequestLicenseAgreement = "REQUEST_LICENSE_AGREEMENT"
	SessionEventStatusTypeError                   = "ERROR"
)

type StatusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type SessionStatusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type CreatedSessionStatusEventBody struct {
}

type StateChangedSessionStatusEventBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type ErrorSessionStatusEventBody struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}
