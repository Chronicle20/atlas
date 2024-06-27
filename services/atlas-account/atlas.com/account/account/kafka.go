package account

import (
	"atlas-account/tenant"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	EnvCommandTopicCreateAccount = "COMMAND_TOPIC_CREATE_ACCOUNT"
	EnvEventTopicAccountStatus   = "EVENT_TOPIC_ACCOUNT_STATUS"
	EventAccountStatusCreated    = "CREATED"
)

type createCommand struct {
	Tenant   tenant.Model `json:"tenant"`
	Name     string       `json:"name"`
	Password string       `json:"password"`
}

type statusEvent struct {
	Tenant    tenant.Model `json:"tenant"`
	AccountId uint32       `json:"account_id"`
	Name      string       `json:"name"`
	Status    string       `json:"status"`
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
