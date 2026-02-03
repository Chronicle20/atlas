package account

const (
	EnvEventTopicStatus  = "EVENT_TOPIC_ACCOUNT_STATUS"
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
