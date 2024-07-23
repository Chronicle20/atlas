package session

import (
	"atlas-channel/tenant"
)

const (
	EnvCommandTopicAccountLogout = "COMMAND_TOPIC_ACCOUNT_LOGOUT"
)

type logoutCommand struct {
	Tenant    tenant.Model `json:"tenant"`
	Issuer    string       `json:"author"`
	AccountId uint32       `json:"accountId"`
}
