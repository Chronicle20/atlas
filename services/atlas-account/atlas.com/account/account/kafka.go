package account

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopicAccountLogout = "COMMAND_TOPIC_ACCOUNT_LOGOUT"
	EnvCommandTopicCreateAccount = "COMMAND_TOPIC_CREATE_ACCOUNT"
	EnvEventTopicAccountStatus   = "EVENT_TOPIC_ACCOUNT_STATUS"
	EventAccountStatusCreated    = "CREATED"
	EventAccountStatusLoggedIn   = "LOGGED_IN"
	EventAccountStatusLoggedOut  = "LOGGED_OUT"
)

type createCommand struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type logoutCommand struct {
	SessionId uuid.UUID `json:"sessionId"`
	Issuer    string    `json:"author"`
	AccountId uint32    `json:"accountId"`
}

type statusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}
