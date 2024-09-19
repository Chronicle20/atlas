package account

const (
	EnvEventTopicAccountStatus  = "EVENT_TOPIC_ACCOUNT_STATUS"
	EventAccountStatusLoggedIn  = "LOGGED_IN"
	EventAccountStatusLoggedOut = "LOGGED_OUT"
)

type statusEvent struct {
	AccountId uint32 `json:"account_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}
