package account

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicAccountStatus topic.Token = "EVENT_TOPIC_ACCOUNT_STATUS"
)

const (
	EventAccountStatusLoggedIn  = "LOGGED_IN"
	EventAccountStatusLoggedOut = "LOGGED_OUT"
)

type StatusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}
