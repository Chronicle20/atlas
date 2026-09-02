package account

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_ACCOUNT_STATUS"
)

const (
	EventStatusCreated   = "CREATED"
	EventStatusLoggedIn  = "LOGGED_IN"
	EventStatusLoggedOut = "LOGGED_OUT"
	EventStatusDeleted   = "DELETED"
)

type StatusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}
