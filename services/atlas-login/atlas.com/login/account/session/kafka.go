package session

import (
	"atlas-login/tenant"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	EnvCommandTopicAccountLogout = "COMMAND_TOPIC_ACCOUNT_LOGOUT"
)

type logoutCommand struct {
	Tenant    tenant.Model `json:"tenant"`
	Issuer    string       `json:"author"`
	AccountId uint32       `json:"accountId"`
}

func lookupTopic(l logrus.FieldLogger) func(token string) string {
	return func(token string) string {
		t, ok := os.LookupEnv(token)
		if !ok {
			l.Warnf("%s environment variable not set. Defaulting to env variable.", token)
			return token

		}
		return t
	}
}
