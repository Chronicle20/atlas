package session

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopicAccountLogout = "COMMAND_TOPIC_ACCOUNT_LOGOUT"
)

type logoutCommand struct {
	SessionId uuid.UUID `json:"sessionId"`
	Issuer    string    `json:"author"`
	AccountId uint32    `json:"accountId"`
}
