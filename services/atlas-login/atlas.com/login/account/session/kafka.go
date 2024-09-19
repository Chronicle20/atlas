package session

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

const (
	EnvCommandTopicAccountLogout = "COMMAND_TOPIC_ACCOUNT_LOGOUT"
)

type logoutCommand struct {
	Tenant    tenant.Model `json:"tenant"`
	Issuer    string       `json:"author"`
	SessionId uuid.UUID    `json:"sessionId"`
	AccountId uint32       `json:"accountId"`
}
