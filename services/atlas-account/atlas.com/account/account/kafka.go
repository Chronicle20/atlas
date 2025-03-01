package account

import "github.com/google/uuid"

const (
	EnvCreateAccountCommandTopic = "COMMAND_TOPIC_CREATE_ACCOUNT"
	EnvSessionCommandTopic       = "COMMAND_TOPIC_ACCOUNT_SESSION"

	SessionCommandIssuerInternal = "INTERNAL"
	SessionCommandTypeLogout     = "LOGOUT"
)

type createCommand struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type accountSessionCommand[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Issuer    string    `json:"author"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type logoutCommandBody struct {
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

type statusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type sessionStatusEvent[E any] struct {
	SessionId uuid.UUID `json:"sessionId"`
	AccountId uint32    `json:"accountId"`
	Type      string    `json:"type"`
	Body      E         `json:"body"`
}

type createdSessionStatusEventBody struct {
}

type stateChangedSessionStatusEventBody struct {
	State  uint8       `json:"state"`
	Params interface{} `json:"params"`
}

type errorSessionStatusEventBody struct {
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}
