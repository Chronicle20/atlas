package session

import (
	"atlas-account/tenant"
)

const (
	EnvCommandTopicAccountLogout = "COMMAND_TOPIC_ACCOUNT_LOGOUT"
	EnvEventTopicAccountStatus   = "EVENT_TOPIC_ACCOUNT_STATUS"
	EventAccountStatusLoggedIn   = "LOGGED_IN"
	EventAccountStatusLoggedOut  = "LOGGED_OUT"
)

type statusEvent struct {
	Tenant    tenant.Model `json:"tenant"`
	AccountId uint32       `json:"account_id"`
	Name      string       `json:"name"`
	Status    string       `json:"status"`
}

type logoutCommand struct {
	Tenant    tenant.Model `json:"tenant"`
	AccountId uint32       `json:"accountId"`
}
